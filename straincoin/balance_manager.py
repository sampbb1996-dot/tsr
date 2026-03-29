"""
Balance Manager — PostgreSQL-backed user accounts, positions, P&L, and fees.

PRINCIPLES
──────────
- Every balance change is atomic (SELECT FOR UPDATE row lock).
- Every balance change is logged in prim_ledger.
- No balance may go negative — enforced at DB level and in code.
- Capital enters ONLY via confirmed Stripe deposits (via webhook).
- Withdrawals are deducted immediately; payout tracked by status field.

TSR ENFORCEMENT (RUNTIME HARD BLOCK)
──────────────────────────────────────
All irreversible methods are prefixed with _ and check _tsr_ctx.active
at entry. If the flag is False (i.e. caller is not inside tsr.commit()),
they raise RuntimeError("TSR_BYPASS_DETECTED") immediately — no DB
operation occurs.

Public methods with the old names exist as tombstones that always raise
RuntimeError. They exist to produce a clear error instead of silently
falling through to an old code path.

THREAD SAFETY
─────────────
psycopg2 connections are per-thread (db.py thread-local). Row-level
locking (SELECT … FOR UPDATE) prevents concurrent balance corruption.
"""

import time
import uuid
from db import execute, execute_in_tx
from tsr import _tsr_ctx

FEE_RATE = 0.001   # 0.1% per trade (buy and sell)

_BYPASS_MSG = "TSR_BYPASS_DETECTED: irreversible method called outside tsr.commit()"


def _require_tsr_context():
    """Raise immediately if not executing inside tsr.commit()."""
    if not getattr(_tsr_ctx, "active", False):
        raise RuntimeError(_BYPASS_MSG)


class BalanceManager:

    # ── Read-only helpers (no TSR gate needed) ────────────────────────────

    def user_exists(self, user_id: str) -> bool:
        rows = execute(
            'SELECT 1 FROM prim_users WHERE id = %s AND is_active = TRUE',
            (user_id,),
        )
        return bool(rows)

    def create_user(self, user_id: str | None = None) -> str:
        """
        Create a bare user record. No money moves — not a financial action.
        Called by auth.register, not exposed publicly.
        """
        uid = user_id or str(uuid.uuid4())
        execute(
            '''
            INSERT INTO prim_users (id, email, password_hash)
            VALUES (%s, %s, '')
            ON CONFLICT DO NOTHING
            ''',
            (uid, f'__anon_{uid}@prim.internal'),
        )
        execute(
            'INSERT INTO prim_balances (user_id, balance) VALUES (%s, 0) ON CONFLICT DO NOTHING',
            (uid,),
        )
        return uid

    def get_balance(self, user_id: str) -> float | None:
        rows = execute(
            'SELECT balance FROM prim_balances WHERE user_id = %s',
            (user_id,),
        )
        return float(rows[0]['balance']) if rows else None

    def get_position(self, user_id: str) -> dict | None:
        rows = execute(
            'SELECT entry_price, size FROM prim_positions WHERE user_id = %s',
            (user_id,),
        )
        if not rows:
            return None
        return {
            'entry_price': float(rows[0]['entry_price']),
            'size':        float(rows[0]['size']),
        }

    # ── Public tombstones — old names now forbidden ───────────────────────
    # These exist so that old call sites produce a clear error rather than
    # silently executing without TSR. They must never be re-implemented.

    def credit_confirmed_deposit(self, *_, **__):
        raise RuntimeError("Direct execution forbidden — use tsr.commit() with _credit_confirmed_deposit")

    def deposit(self, *_, **__):
        raise RuntimeError("Direct execution forbidden — use tsr.commit() with _deposit")

    def withdraw(self, *_, **__):
        raise RuntimeError("Direct execution forbidden — use tsr.commit() with _withdraw")

    def restore_withdrawal(self, *_, **__):
        raise RuntimeError("Direct execution forbidden — use tsr.commit() with _restore_withdrawal")

    def execute_buy(self, *_, **__):
        raise RuntimeError("Direct execution forbidden — use tsr.commit() with _execute_buy")

    def execute_sell(self, *_, **__):
        raise RuntimeError("Direct execution forbidden — use tsr.commit() with _execute_sell")

    # ── PRIVATE irreversible mutations ────────────────────────────────────
    # Each method calls _require_tsr_context() as its first statement.
    # No DB write can occur unless _tsr_ctx.active is True.

    def _credit_confirmed_deposit(self, user_id: str, amount: float,
                                  deposit_id: str, stripe_pi_id: str) -> dict:
        _require_tsr_context()

        if amount <= 0:
            return {'ok': False, 'reason': 'amount must be positive'}

        try:
            rows = execute(
                '''
                WITH claim AS (
                    UPDATE prim_deposits
                       SET status       = 'confirmed',
                           confirmed_at = NOW()
                     WHERE id     = %s
                       AND status = 'pending'
                    RETURNING id
                ),
                credit AS (
                    UPDATE prim_balances
                       SET balance    = balance + %s,
                           updated_at = NOW()
                     WHERE user_id = %s
                       AND EXISTS (SELECT 1 FROM claim)
                    RETURNING balance
                ),
                ledger AS (
                    INSERT INTO prim_ledger
                        (user_id, type, amount, balance_after, ref_id)
                    SELECT %s, 'deposit', %s, balance, %s
                      FROM credit
                    RETURNING 1
                )
                SELECT
                    (SELECT balance FROM credit)         AS new_balance,
                    (SELECT COUNT(*) FROM claim)::int    AS claimed
                ''',
                (deposit_id, amount, user_id, user_id, amount, stripe_pi_id),
            )
            row     = rows[0] if rows else {}
            claimed = row.get('claimed', 0)

            if not claimed:
                bal_row = execute(
                    'SELECT balance FROM prim_balances WHERE user_id = %s', (user_id,)
                )
                balance = float(bal_row[0]['balance']) if bal_row else 0.0
                return {'ok': True, 'balance': round(balance, 6), 'already_credited': True}

            new_balance = float(row.get('new_balance') or 0)
            return {'ok': True, 'balance': round(new_balance, 6), 'already_credited': False}
        except Exception as e:
            return {'ok': False, 'reason': str(e)}

    def _deposit(self, user_id: str, amount: float) -> dict:
        _require_tsr_context()

        if amount <= 0:
            return {'ok': False, 'reason': 'amount must be positive'}
        rows = execute(
            'SELECT 1 FROM prim_users WHERE id = %s AND is_active = TRUE', (user_id,)
        )
        if not rows:
            return {'ok': False, 'reason': 'user not found'}

        deposit_id = str(uuid.uuid4())
        try:
            results = execute_in_tx([
                (
                    'SELECT balance FROM prim_balances WHERE user_id = %s FOR UPDATE',
                    (user_id,),
                ),
                (
                    '''
                    UPDATE prim_balances
                       SET balance = balance + %s, updated_at = NOW()
                     WHERE user_id = %s
                    RETURNING balance
                    ''',
                    (amount, user_id),
                ),
                (
                    '''
                    INSERT INTO prim_deposits
                        (id, user_id, amount, status, confirmed_at)
                    VALUES (%s, %s, %s, 'confirmed', NOW())
                    ''',
                    (deposit_id, user_id, amount),
                ),
                (
                    '''
                    INSERT INTO prim_ledger (user_id, type, amount, balance_after, ref_id)
                    SELECT %s, 'deposit', %s, balance, %s
                      FROM prim_balances WHERE user_id = %s
                    ''',
                    (user_id, amount, deposit_id, user_id),
                ),
            ])
            new_bal = float(results[1][0]['balance'])
            return {
                'ok':      True,
                'user_id': user_id,
                'amount':  round(amount, 6),
                'balance': round(new_bal, 6),
                'record':  {
                    'user_id':   user_id,
                    'amount':    round(amount, 6),
                    'timestamp': time.time(),
                    'status':    'confirmed',
                },
            }
        except Exception as e:
            return {'ok': False, 'reason': str(e)}

    def _withdraw(self, user_id: str, amount: float,
                  destination: str | None = None) -> dict:
        _require_tsr_context()

        if amount <= 0:
            return {'ok': False, 'reason': 'amount must be positive'}
        wd_id = str(uuid.uuid4())
        sql = '''
        WITH deduct AS (
            UPDATE prim_balances
               SET balance    = balance - %(amount)s,
                   updated_at = NOW()
             WHERE user_id = %(uid)s
               AND balance  >= %(amount)s
            RETURNING balance
        ),
        ins_wd AS (
            INSERT INTO prim_withdrawals (id, user_id, amount, destination, status)
            SELECT %(wd_id)s, %(uid)s, %(amount)s, %(dest)s, 'pending'
              FROM deduct
        ),
        ins_ledger AS (
            INSERT INTO prim_ledger (user_id, type, amount, balance_after, ref_id)
            SELECT %(uid)s, 'withdrawal', -%(amount)s, balance, %(wd_id)s
              FROM deduct
        )
        SELECT balance FROM deduct
        '''
        try:
            rows = execute(sql, {
                'uid': user_id, 'amount': amount,
                'dest': destination or '', 'wd_id': wd_id,
            })
        except Exception as e:
            return {'ok': False, 'reason': str(e)}

        if not rows:
            bal_rows = execute(
                'SELECT balance FROM prim_balances WHERE user_id = %s', (user_id,)
            )
            if not bal_rows:
                return {'ok': False, 'reason': 'user not found'}
            current = float(bal_rows[0]['balance'])
            return {'ok': False, 'reason': f'insufficient balance ({current:.4f} < {amount:.4f})'}

        new_bal = float(rows[0]['balance'])
        return {
            'ok':      True,
            'user_id': user_id,
            'amount':  round(amount, 6),
            'balance': round(new_bal, 6),
            'record':  {
                'id':        wd_id,
                'user_id':   user_id,
                'amount':    round(amount, 6),
                'timestamp': time.time(),
                'status':    'pending',
            },
        }

    def _restore_withdrawal(self, withdrawal_id: str) -> dict:
        _require_tsr_context()

        rows = execute(
            'SELECT user_id, amount FROM prim_withdrawals WHERE id = %s',
            (withdrawal_id,),
        )
        if not rows:
            return {'ok': False, 'reason': 'withdrawal not found'}

        user_id = str(rows[0]['user_id'])
        amount  = float(rows[0]['amount'])
        ref     = str(uuid.uuid4())

        try:
            execute_in_tx([
                (
                    '''
                    UPDATE prim_balances
                       SET balance = balance + %s, updated_at = NOW()
                     WHERE user_id = %s
                    ''',
                    (amount, user_id),
                ),
                (
                    "UPDATE prim_withdrawals SET status = 'failed' WHERE id = %s",
                    (withdrawal_id,),
                ),
                (
                    '''
                    INSERT INTO prim_ledger (user_id, type, amount, balance_after, ref_id)
                    SELECT %s, 'deposit', %s, balance, %s
                      FROM prim_balances WHERE user_id = %s
                    ''',
                    (user_id, amount, ref, user_id),
                ),
            ])
            return {'ok': True, 'user_id': user_id, 'amount': amount}
        except Exception as e:
            return {'ok': False, 'reason': str(e)}

    def _execute_buy(self, user_id: str, size: float, price: float) -> dict:
        _require_tsr_context()

        cost  = round(size * price, 8)
        fee   = round(cost * FEE_RATE, 8)
        total = round(cost + fee, 8)

        try:
            results = execute_in_tx([
                (
                    'SELECT balance FROM prim_balances WHERE user_id = %s FOR UPDATE',
                    (user_id,),
                ),
                (
                    '''
                    UPDATE prim_balances
                       SET balance = balance - %s, updated_at = NOW()
                     WHERE user_id = %s AND balance >= %s
                    RETURNING balance
                    ''',
                    (total, user_id, total),
                ),
                (
                    '''
                    INSERT INTO prim_positions (user_id, entry_price, size)
                    VALUES (%s, %s, %s)
                    ON CONFLICT (user_id) DO UPDATE
                       SET entry_price = EXCLUDED.entry_price,
                           size        = prim_positions.size + EXCLUDED.size
                    RETURNING entry_price, size
                    ''',
                    (user_id, price, size),
                ),
                (
                    '''
                    INSERT INTO prim_ledger (user_id, type, amount, balance_after, ref_id)
                    SELECT %s, 'buy', -%s, balance, gen_random_uuid()
                      FROM prim_balances WHERE user_id = %s
                    ''',
                    (user_id, total, user_id),
                ),
            ])
            if not results[1]:
                bal = execute(
                    'SELECT balance FROM prim_balances WHERE user_id = %s', (user_id,)
                )
                current = float(bal[0]['balance']) if bal else 0.0
                return {'ok': False, 'reason': f'insufficient balance ({current:.4f} < {total:.4f})'}

            new_bal  = float(results[1][0]['balance'])
            position = results[2][0] if results[2] else {}
            return {
                'ok':       True,
                'balance':  round(new_bal, 6),
                'cost':     round(cost, 6),
                'fee':      round(fee, 6),
                'position': {
                    'entry_price': float(position.get('entry_price', price)),
                    'size':        float(position.get('size', size)),
                },
            }
        except Exception as e:
            return {'ok': False, 'reason': str(e)}

    def _execute_sell(self, user_id: str, size: float, price: float) -> dict:
        _require_tsr_context()

        proceeds = round(size * price, 8)
        fee      = round(proceeds * FEE_RATE, 8)
        net      = round(proceeds - fee, 8)

        try:
            results = execute_in_tx([
                (
                    'SELECT entry_price, size FROM prim_positions WHERE user_id = %s FOR UPDATE',
                    (user_id,),
                ),
                (
                    '''
                    UPDATE prim_positions
                       SET size = size - %s
                     WHERE user_id = %s AND size >= %s
                    RETURNING size
                    ''',
                    (size, user_id, size),
                ),
                (
                    '''
                    UPDATE prim_balances
                       SET balance = balance + %s, updated_at = NOW()
                     WHERE user_id = %s
                    RETURNING balance
                    ''',
                    (net, user_id),
                ),
                (
                    '''
                    INSERT INTO prim_ledger (user_id, type, amount, balance_after, ref_id)
                    SELECT %s, 'sell', %s, balance, gen_random_uuid()
                      FROM prim_balances WHERE user_id = %s
                    ''',
                    (user_id, net, user_id),
                ),
            ])
            if not results[1]:
                pos = execute(
                    'SELECT size FROM prim_positions WHERE user_id = %s', (user_id,)
                )
                current_size = float(pos[0]['size']) if pos else 0.0
                return {
                    'ok':     False,
                    'reason': f'insufficient position ({current_size:.6f} < {size:.6f})',
                }

            new_bal      = float(results[2][0]['balance']) if results[2] else 0.0
            new_pos_size = float(results[1][0]['size']) if results[1] else 0.0
            return {
                'ok':       True,
                'balance':  round(new_bal, 6),
                'proceeds': round(proceeds, 6),
                'fee':      round(fee, 6),
                'position': {'size': round(new_pos_size, 6)},
            }
        except Exception as e:
            return {'ok': False, 'reason': str(e)}

    # ── Status-only mutations (no money moves — no TSR gate required) ──────

    def update_withdrawal_payout(self, withdrawal_id: str,
                                 stripe_payout_id: str) -> None:
        """Mark withdrawal as 'processing'. Status change only."""
        execute(
            '''
            UPDATE prim_withdrawals
               SET stripe_payout_id = %s, status = 'processing'
             WHERE id = %s
            ''',
            (stripe_payout_id, withdrawal_id),
        )

    def complete_withdrawal(self, withdrawal_id: str) -> None:
        """Mark withdrawal as 'completed'. Status change only."""
        execute(
            '''
            UPDATE prim_withdrawals
               SET status = 'completed', completed_at = NOW()
             WHERE id = %s
            ''',
            (withdrawal_id,),
        )

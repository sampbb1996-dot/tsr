"""
TSR-ENFORCED ROUTE IMPLEMENTATIONS
====================================

Drop-in replacements for the irreversible API routes in dashboard.py.

Every irreversible action passes through tsr.commit().
No balance mutation, trade execution, or withdrawal can occur without it.

To apply: replace the corresponding route functions in dashboard.py with
the implementations below. The rest of dashboard.py is unchanged.

Prerequisites in dashboard.py (already present after earlier patch):
    from tsr import TSR, TSRBlocked
    tsr = TSR()

After app.py injects balance_manager:
    tsr.balance_manager = balance_manager  # enables balance pre-check in validate()
"""

# ─────────────────────────────────────────────────────────────────────────────
# TRADE  —  POST /api/trade
# BEFORE: called balance_manager.execute_buy/sell directly
# AFTER:  all execution is downstream of tsr.commit()
# ─────────────────────────────────────────────────────────────────────────────

@app.route('/api/trade', methods=['POST'])
@require_auth
def api_trade(user_id):
    if balance_manager is None:
        return jsonify({'error': 'service unavailable'}), 503

    data = request.get_json(silent=True) or {}
    side = data.get('side', '').lower()
    size = data.get('size')

    if side not in ('buy', 'sell'):
        return jsonify({'error': 'side must be buy or sell'}), 400
    try:
        size = float(size)
        if size <= 0:
            raise ValueError
    except (TypeError, ValueError):
        return jsonify({'error': 'size must be a positive number'}), 400

    # Get current market price from exchange
    if exchange is None:
        return jsonify({'error': 'exchange not ready'}), 503
    price = exchange.get_mid_price()
    if price is None or price <= 0:
        return jsonify({'error': 'no market price available'}), 503

    action = {
        'type':    'trade',
        'user_id': user_id,
        'amount':  size,
        'meta': {
            'side':           side,
            'price':          price,
            'fee_rate':       0.001,
            'min_order_size': 0.0001,
        },
    }

    try:
        if side == 'buy':
            result = tsr.commit(action, lambda: balance_manager._execute_buy(user_id, size, price))
        else:
            result = tsr.commit(action, lambda: balance_manager._execute_sell(user_id, size, price))
    except TSRBlocked as e:
        return jsonify(e.to_dict()), 403

    if not result.get('ok'):
        return jsonify({'error': result.get('reason', 'trade failed')}), 400

    return jsonify(result)


# ─────────────────────────────────────────────────────────────────────────────
# WITHDRAW  —  POST /api/withdraw
# BEFORE: called balance_manager.withdraw directly
# AFTER:  all execution is downstream of tsr.commit()
# ─────────────────────────────────────────────────────────────────────────────

@app.route('/api/withdraw', methods=['POST'])
@require_auth
def api_withdraw(user_id):
    if balance_manager is None:
        return jsonify({'error': 'service unavailable'}), 503

    data        = request.get_json(silent=True) or {}
    amount      = data.get('amount')
    destination = data.get('destination', '')

    try:
        amount = float(amount)
        if amount <= 0:
            raise ValueError
    except (TypeError, ValueError):
        return jsonify({'error': 'amount must be a positive number'}), 400

    action = {
        'type':    'withdraw',
        'user_id': user_id,
        'amount':  amount,
        'meta': {
            'destination': destination,
            'fee_rate':    0.001,
        },
    }

    try:
        result = tsr.commit(action, lambda: balance_manager._withdraw(user_id, amount, destination))
    except TSRBlocked as e:
        return jsonify(e.to_dict()), 403

    if not result.get('ok'):
        return jsonify({'error': result.get('reason', 'withdrawal failed')}), 400

    return jsonify(result)


# ─────────────────────────────────────────────────────────────────────────────
# STAKE WITHDRAW  —  POST /api/stake/withdraw
# Same enforcement as /api/withdraw
# ─────────────────────────────────────────────────────────────────────────────

@app.route('/api/stake/withdraw', methods=['POST'])
@require_auth
def api_stake_withdraw(user_id):
    if balance_manager is None:
        return jsonify({'error': 'service unavailable'}), 503

    data        = request.get_json(silent=True) or {}
    amount      = data.get('amount')
    destination = data.get('destination', '')

    try:
        amount = float(amount)
        if amount <= 0:
            raise ValueError
    except (TypeError, ValueError):
        return jsonify({'error': 'amount must be a positive number'}), 400

    action = {
        'type':    'withdraw',
        'user_id': user_id,
        'amount':  amount,
        'meta':    {'destination': destination, 'fee_rate': 0.001},
    }

    try:
        result = tsr.commit(action, lambda: balance_manager._withdraw(user_id, amount, destination))
    except TSRBlocked as e:
        return jsonify(e.to_dict()), 403

    if not result.get('ok'):
        return jsonify({'error': result.get('reason', 'withdrawal failed')}), 400

    return jsonify(result)


# ─────────────────────────────────────────────────────────────────────────────
# STRIPE WEBHOOK  —  POST /api/stripe/webhook
# BEFORE: called balance_manager.credit_confirmed_deposit directly
# AFTER:  credit only occurs after tsr.commit() — READONLY blocks it entirely
# ─────────────────────────────────────────────────────────────────────────────

@app.route('/api/stripe/webhook', methods=['POST'])
def api_stripe_webhook():
    payload    = request.get_data()
    sig_header = request.headers.get('Stripe-Signature', '')

    try:
        event = stripe.Webhook.construct_event(payload, sig_header, _STRIPE_WEBHOOK_SECRET)
    except stripe.error.SignatureVerificationError:
        return jsonify({'error': 'invalid signature'}), 400
    except Exception:
        return jsonify({'error': 'webhook error'}), 400

    # ── Payment confirmed ─────────────────────────────────────────────────
    if event['type'] == 'checkout.session.completed':
        session    = event['data']['object']
        user_id    = session.get('metadata', {}).get('user_id')
        deposit_id = session.get('metadata', {}).get('deposit_id')
        amount_raw = session.get('amount_total', 0)

        if not user_id or not deposit_id or not amount_raw:
            return jsonify({'error': 'missing metadata'}), 400

        amount     = round(float(amount_raw) / 100, 6)   # Stripe amounts in cents
        stripe_pi  = session.get('payment_intent', '')

        if balance_manager is None:
            return jsonify({'error': 'balance manager not ready'}), 503

        action = {
            'type':    'deposit_credit',
            'user_id': user_id,
            'amount':  amount,
            'meta': {
                'deposit_id':   deposit_id,
                'stripe_pi_id': stripe_pi,
            },
        }

        try:
            result = tsr.commit(
                action,
                lambda: balance_manager._credit_confirmed_deposit(
                    user_id, amount, deposit_id, stripe_pi
                ),
            )
        except TSRBlocked as e:
            app.logger.error(f'[TSR] deposit credit blocked: {e.reason} user={user_id}')
            # Return 200 to Stripe so it does not retry — the block is intentional.
            return jsonify({'blocked': True, 'reason': e.reason}), 200

        if not result.get('ok'):
            app.logger.error(f'deposit credit failed: {result.get("reason")} user={user_id}')
            return jsonify({'error': result.get('reason')}), 500

    # ── Payout failed — restore balance ──────────────────────────────────
    elif event['type'] == 'payout.failed':
        payout        = event['data']['object']
        withdrawal_id = payout.get('metadata', {}).get('withdrawal_id')

        if withdrawal_id and balance_manager is not None:
            # Look up user_id + amount for TSR action dict
            rows = execute(
                'SELECT user_id, amount FROM prim_withdrawals WHERE id = %s',
                (withdrawal_id,),
            )
            if rows:
                uid = str(rows[0]['user_id'])
                amt = float(rows[0]['amount'])
                action = {
                    'type':    'restore',
                    'user_id': uid,
                    'amount':  amt,
                    'meta':    {'withdrawal_id': withdrawal_id},
                }
                try:
                    tsr.commit(action, lambda: balance_manager._restore_withdrawal(withdrawal_id))
                except TSRBlocked as e:
                    app.logger.error(f'[TSR] withdrawal restore blocked: {e.reason}')

    elif event['type'] == 'payout.paid':
        payout        = event['data']['object']
        withdrawal_id = payout.get('metadata', {}).get('withdrawal_id')
        if withdrawal_id and balance_manager is not None:
            # Status-only update — no money moves, no TSR gate required
            balance_manager.complete_withdrawal(withdrawal_id)

    return jsonify({'ok': True})


# ─────────────────────────────────────────────────────────────────────────────
# ADMIN CREDIT  —  POST /api/admin/credit
# BEFORE: called balance_manager.deposit directly
# AFTER:  all execution is downstream of tsr.commit()
# ─────────────────────────────────────────────────────────────────────────────

_ADMIN_SECRET = os.environ.get('ADMIN_SECRET', '')

@app.route('/api/admin/credit', methods=['POST'])
def api_admin_credit():
    secret = request.headers.get('X-Admin-Secret', '')
    if not _ADMIN_SECRET or secret != _ADMIN_SECRET:
        return jsonify({'error': 'forbidden'}), 403

    if balance_manager is None:
        return jsonify({'error': 'service unavailable'}), 503

    data    = request.get_json(silent=True) or {}
    user_id = data.get('user_id', '').strip()
    amount  = data.get('amount')
    reason  = data.get('reason', 'admin credit')

    if not user_id:
        return jsonify({'error': 'user_id required'}), 400
    try:
        amount = float(amount)
        if amount <= 0:
            raise ValueError
    except (TypeError, ValueError):
        return jsonify({'error': 'amount must be a positive number'}), 400

    action = {
        'type':    'admin_credit',
        'user_id': user_id,
        'amount':  amount,
        'meta':    {'reason': reason},
    }

    try:
        result = tsr.commit(action, lambda: balance_manager._deposit(user_id, amount))
    except TSRBlocked as e:
        return jsonify(e.to_dict()), 403

    if not result.get('ok'):
        return jsonify({'error': result.get('reason', 'credit failed')}), 400

    return jsonify(result)


# ─────────────────────────────────────────────────────────────────────────────
# TSR REGIME CONTROL  —  GET/POST /api/tsr/regime
# ─────────────────────────────────────────────────────────────────────────────

@app.route('/api/tsr/regime', methods=['GET'])
def api_tsr_regime_get():
    return jsonify(tsr.status())


@app.route('/api/tsr/regime', methods=['POST'])
def api_tsr_regime_set():
    body   = request.get_json(force=True) or {}
    regime = body.get('regime', '').upper()
    if regime not in ('CALM', 'READONLY'):
        return jsonify({'error': 'regime must be CALM or READONLY'}), 400
    tsr.set_regime(regime)
    return jsonify(tsr.status())


# ─────────────────────────────────────────────────────────────────────────────
# app.py ADDITION — inject tsr.balance_manager after engine init
# Add this line inside _init_engines(), after: dash_module.balance_manager = balances
# ─────────────────────────────────────────────────────────────────────────────
#
#   dash_module.tsr.balance_manager = balances
#
# This gives TSR.validate() access to get_balance() for pre-commit balance checks.
# ─────────────────────────────────────────────────────────────────────────────

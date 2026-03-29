"""
TSR — Execution Governance Layer for PRIM
==========================================

Sits between decision logic and execution.
No irreversible action (trade, withdrawal, balance credit) may occur
without passing through TSR.commit().

Regimes
───────
  CALM     — irreversible actions allowed inside commit()
  READONLY — all irreversible actions blocked, commit() raises TSRBlocked

Runtime guard
─────────────
  _tsr_ctx.active is a thread-local flag set to True only while
  TSR.commit() is executing execute_fn(). Every private mutation method
  in BalanceManager checks this flag and raises RuntimeError if False.
  This makes bypass structurally impossible at runtime.
"""

import threading
import logging

logger = logging.getLogger(__name__)

# ── Thread-local execution context ────────────────────────────────────────────
# Set to True only while TSR.commit() is executing execute_fn().
# BalanceManager._private methods check this flag — any call outside
# TSR.commit() raises RuntimeError("TSR_BYPASS_DETECTED") immediately.
_tsr_ctx = threading.local()


class TSRBlocked(Exception):
    """
    Raised by TSR.commit() when an action fails validation.

    Attributes
    ──────────
    reason   — human-readable explanation
    action   — the action dict that was rejected
    """
    def __init__(self, reason: str, action: dict | None = None):
        super().__init__(reason)
        self.reason = reason
        self.action = action or {}

    def to_dict(self) -> dict:
        return {"error": "blocked_by_tsr", "reason": self.reason}


class TSR:

    def __init__(self):
        self._lock   = threading.Lock()
        self._regime = "CALM"
        self._audit: list[dict] = []
        # Injected by app.py after engine init:
        #   dash_module.tsr.balance_manager = balances
        # Enables validate() to check available balance before any commit.
        self.balance_manager = None

    # ── Regime control ────────────────────────────────────────────────────

    def set_regime(self, regime: str) -> None:
        assert regime in ("CALM", "READONLY"), f"Unknown regime: {regime!r}"
        with self._lock:
            prev = self._regime
            self._regime = regime
        logger.info(f"[TSR] regime changed: {prev} → {regime}")

    def get_regime(self) -> str:
        with self._lock:
            return self._regime

    # ── Validation — single source of truth ──────────────────────────────

    def validate(self, action: dict) -> tuple[bool, str]:
        """
        Validate an action before execution. This is the ONLY place
        where action constraints are checked — do not duplicate elsewhere.

        action = {
            "type":    "trade" | "withdraw" | "deposit_credit" |
                       "restore" | "admin_credit",
            "user_id": str,
            "amount":  float,
            "meta":    dict  (optional)
        }

        Returns (True, "") on pass, (False, reason) on failure.
        """
        with self._lock:
            regime = self._regime

        action_type = action.get("type", "unknown")
        user_id     = action.get("user_id")
        amount      = action.get("amount", 0)
        meta        = action.get("meta", {}) or {}

        # 1. READONLY blocks all irreversible actions without exception
        if regime == "READONLY":
            reason = f"regime=READONLY blocks all irreversible actions (type={action_type})"
            logger.warning(f"[TSR] BLOCKED — {reason} user={user_id}")
            return False, reason

        # 2. Amount must be strictly positive
        if not isinstance(amount, (int, float)) or amount <= 0:
            reason = f"amount must be > 0, got {amount!r}"
            logger.warning(f"[TSR] BLOCKED — {reason} user={user_id}")
            return False, reason

        # 3. Balance check for debit operations (trade, withdraw)
        if action_type in ("withdraw", "trade") and self.balance_manager is not None and user_id:
            available = self.balance_manager.get_balance(user_id)
            if available is None:
                reason = "user account not found"
                logger.warning(f"[TSR] BLOCKED — {reason} user={user_id}")
                return False, reason

            # Worst-case fees are accounted for BEFORE allowing execution
            fee_rate   = float(meta.get("fee_rate", 0.001))
            total_cost = amount + (amount * fee_rate)

            if total_cost > available:
                reason = (
                    f"insufficient balance: need {total_cost:.6f} "
                    f"(amount={amount} + fee={amount * fee_rate:.6f}), "
                    f"available={available:.6f}"
                )
                logger.warning(f"[TSR] BLOCKED — {reason} user={user_id}")
                return False, reason

        # 4. Minimum order size for trades — skip (do NOT clamp), never execute below minimum
        if action_type == "trade":
            min_size = meta.get("min_order_size")
            if min_size is not None and amount < float(min_size):
                reason = f"trade size {amount} is below minimum order size {min_size}"
                logger.warning(f"[TSR] BLOCKED — {reason} user={user_id}")
                return False, reason

        return True, ""

    # ── Commit gate ───────────────────────────────────────────────────────

    def commit(self, action: dict, execute_fn):
        """
        The only path through which irreversible actions may execute.

        Flow
        ────
        1. validate(action) — raises TSRBlocked on failure
        2. Set _tsr_ctx.active = True  (thread-local execution flag)
        3. Call execute_fn()            (the irreversible operation)
        4. Clear _tsr_ctx.active = False in finally (always)
        5. Append to audit log
        6. Return result

        The thread-local flag in step 2-4 is the runtime enforcement
        mechanism.  BalanceManager._private methods check this flag at
        entry and raise RuntimeError("TSR_BYPASS_DETECTED") if False.
        """
        ok, reason = self.validate(action)
        if not ok:
            raise TSRBlocked(reason, action)

        _tsr_ctx.active = True
        try:
            result = execute_fn()
        finally:
            _tsr_ctx.active = False

        with self._lock:
            self._audit.append({
                "type":    action.get("type"),
                "user_id": action.get("user_id"),
                "amount":  action.get("amount"),
                "regime":  self._regime,
            })
            if len(self._audit) > 10_000:
                self._audit = self._audit[-5_000:]

        logger.info(
            f"[TSR] COMMITTED — {action.get('type')} "
            f"user={action.get('user_id')} amount={action.get('amount')}"
        )
        return result

    # ── Audit log ─────────────────────────────────────────────────────────

    def get_audit(self, n: int = 100) -> list[dict]:
        with self._lock:
            return list(self._audit[-n:])

    # ── Status ────────────────────────────────────────────────────────────

    def status(self) -> dict:
        with self._lock:
            return {
                "regime":      self._regime,
                "audit_count": len(self._audit),
            }

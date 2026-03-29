"""
TSR — Execution Governance Layer for PRIM
==========================================

Sits between decision logic and execution.
No irreversible action (trade, withdrawal, balance credit) may occur
without passing through TSR.commit().

Regimes
───────
  CALM     — irreversible actions allowed inside commit()
  READONLY — all irreversible actions blocked, commit() raises TSR_BLOCKED
"""

import threading
import logging

logger = logging.getLogger(__name__)


class TSR:

    def __init__(self):
        self._lock   = threading.Lock()
        self._regime = "CALM"
        self._audit: list[dict] = []

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

    # ── Validation ────────────────────────────────────────────────────────

    def validate(self, action: dict) -> bool:
        """
        action = {
            "type":    "trade" | "withdraw" | "deposit",
            "user_id": str,
            "amount":  float,
            "meta":    dict   (optional)
        }
        Returns True if the action is permitted, False otherwise.
        """
        with self._lock:
            regime = self._regime

        if regime == "READONLY":
            logger.warning(f"[TSR] BLOCKED (READONLY) — {action.get('type')} "
                           f"user={action.get('user_id')} amount={action.get('amount')}")
            return False

        if action.get("amount", 0) <= 0:
            logger.warning(f"[TSR] BLOCKED (amount <= 0) — {action}")
            return False

        return True

    # ── Commit gate ───────────────────────────────────────────────────────

    def commit(self, action: dict, execute_fn):
        """
        Gate an irreversible action.

        Parameters
        ──────────
        action      — dict describing the action (see validate)
        execute_fn  — zero-argument callable that performs the real action

        Returns the result of execute_fn() on success.
        Raises Exception("TSR_BLOCKED") if validation fails.
        """
        if not self.validate(action):
            raise Exception("TSR_BLOCKED")

        result = execute_fn()

        with self._lock:
            self._audit.append({
                "type":    action.get("type"),
                "user_id": action.get("user_id"),
                "amount":  action.get("amount"),
                "regime":  self._regime,
            })
            if len(self._audit) > 10_000:
                self._audit = self._audit[-5_000:]

        logger.info(f"[TSR] COMMITTED — {action.get('type')} "
                    f"user={action.get('user_id')} amount={action.get('amount')}")
        return result

    # ── Audit log ─────────────────────────────────────────────────────────

    def get_audit(self, n: int = 100) -> list[dict]:
        with self._lock:
            return list(self._audit[-n:])

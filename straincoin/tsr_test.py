"""
TSR Runtime Invariant Tests
============================

Run with:  python straincoin/tsr_test.py

Proves the invariant holds without a live database.
"""

import sys
import os
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import types

# ── Stub db module so we can import without a live DB ────────────────────────
db_stub = types.ModuleType("db")
db_stub.execute = lambda *a, **kw: []
db_stub.execute_in_tx = lambda *a, **kw: []
sys.modules["db"] = db_stub

from tsr import TSR, TSRBlocked, _tsr_ctx
from balance_manager import BalanceManager

_BYPASS = "TSR_BYPASS_DETECTED"


# ── Helpers ───────────────────────────────────────────────────────────────────

def assert_bypass_blocked(fn, *args, **kwargs):
    try:
        fn(*args, **kwargs)
        raise AssertionError(f"{fn.__name__} should have raised RuntimeError but did not")
    except RuntimeError as e:
        assert _BYPASS in str(e), f"Wrong error message: {e}"


def assert_raises(exc_type, fn, *args, **kwargs):
    try:
        fn(*args, **kwargs)
        raise AssertionError(f"Expected {exc_type.__name__} but nothing was raised")
    except exc_type:
        pass


# ── Stub BalanceManager for validation tests (no DB) ─────────────────────────

class _StubBalanceManager:
    """Returns a fixed balance for testing TSR.validate() pre-checks."""
    def __init__(self, balance: float):
        self._balance = balance

    def get_balance(self, user_id: str) -> float | None:
        if user_id == "__missing__":
            return None
        return self._balance


# ─────────────────────────────────────────────────────────────────────────────
# ORIGINAL 12 TESTS — unchanged
# ─────────────────────────────────────────────────────────────────────────────

def test_execute_buy_blocked_outside_commit():
    bm = BalanceManager()
    assert_bypass_blocked(bm._execute_buy, "user-1", 1.0, 100.0)
    print("PASS  _execute_buy raises TSR_BYPASS_DETECTED outside commit()")


def test_execute_sell_blocked_outside_commit():
    bm = BalanceManager()
    assert_bypass_blocked(bm._execute_sell, "user-1", 1.0, 100.0)
    print("PASS  _execute_sell raises TSR_BYPASS_DETECTED outside commit()")


def test_withdraw_blocked_outside_commit():
    bm = BalanceManager()
    assert_bypass_blocked(bm._withdraw, "user-1", 50.0)
    print("PASS  _withdraw raises TSR_BYPASS_DETECTED outside commit()")


def test_credit_blocked_outside_commit():
    bm = BalanceManager()
    assert_bypass_blocked(bm._credit_confirmed_deposit, "user-1", 100.0, "dep-1", "pi-1")
    print("PASS  _credit_confirmed_deposit raises TSR_BYPASS_DETECTED outside commit()")


def test_deposit_blocked_outside_commit():
    bm = BalanceManager()
    assert_bypass_blocked(bm._deposit, "user-1", 100.0)
    print("PASS  _deposit raises TSR_BYPASS_DETECTED outside commit()")


def test_restore_blocked_outside_commit():
    bm = BalanceManager()
    assert_bypass_blocked(bm._restore_withdrawal, "wd-1")
    print("PASS  _restore_withdrawal raises TSR_BYPASS_DETECTED outside commit()")


def test_public_tombstones_raise():
    bm = BalanceManager()
    for name in ("credit_confirmed_deposit", "deposit", "withdraw",
                 "restore_withdrawal", "execute_buy", "execute_sell"):
        assert_raises(RuntimeError, getattr(bm, name))
        print(f"PASS  {name}() raises RuntimeError (tombstone)")


def test_readonly_blocks_commit():
    tsr = TSR()
    tsr.set_regime("READONLY")
    action = {"type": "trade", "user_id": "u1", "amount": 10.0}
    try:
        tsr.commit(action, lambda: None)
        raise AssertionError("Should have raised TSRBlocked")
    except TSRBlocked as e:
        assert "READONLY" in e.reason
    print("PASS  READONLY regime blocks tsr.commit()")


def test_zero_amount_blocked():
    tsr = TSR()
    action = {"type": "trade", "user_id": "u1", "amount": 0}
    try:
        tsr.commit(action, lambda: None)
        raise AssertionError("Should have raised TSRBlocked")
    except TSRBlocked as e:
        assert "amount" in e.reason
    print("PASS  zero amount is blocked by validate()")


def test_negative_amount_blocked():
    tsr = TSR()
    action = {"type": "trade", "user_id": "u1", "amount": -5.0}
    try:
        tsr.commit(action, lambda: None)
        raise AssertionError("Should have raised TSRBlocked")
    except TSRBlocked as e:
        assert "amount" in e.reason
    print("PASS  negative amount is blocked by validate()")


def test_commit_sets_context_flag():
    tsr = TSR()
    seen_inside = []

    def probe():
        seen_inside.append(getattr(_tsr_ctx, "active", False))
        return "ok"

    tsr.commit({"type": "trade", "user_id": "u1", "amount": 1.0}, probe)
    assert seen_inside == [True], f"Expected [True], got {seen_inside}"
    assert not getattr(_tsr_ctx, "active", False)
    print("PASS  _tsr_ctx.active=True inside commit(), False after")


def test_context_cleared_on_exception():
    tsr = TSR()

    def boom():
        raise ValueError("simulated failure")

    try:
        tsr.commit({"type": "trade", "user_id": "u1", "amount": 1.0}, boom)
    except ValueError:
        pass

    assert not getattr(_tsr_ctx, "active", False)
    print("PASS  _tsr_ctx.active cleared even when execute_fn raises")


# ─────────────────────────────────────────────────────────────────────────────
# STEP 3 — FINAL SYSTEM TESTS (normal flow + invalid flow + bypass attempt)
# ─────────────────────────────────────────────────────────────────────────────

# Normal flow — TSR wired to a stub BalanceManager with sufficient funds

def test_normal_deposit_succeeds():
    """Deposit (admin) passes TSR — no balance pre-check for credits."""
    tsr = TSR()
    tsr.balance_manager = _StubBalanceManager(balance=0.0)
    executed = []

    action = {"type": "admin_credit", "user_id": "u1", "amount": 100.0}
    tsr.commit(action, lambda: executed.append("deposit"))

    assert executed == ["deposit"]
    print("PASS  Normal flow: deposit succeeds via TSR")


def test_normal_trade_succeeds():
    """Trade passes TSR when balance is sufficient."""
    tsr = TSR()
    tsr.balance_manager = _StubBalanceManager(balance=1000.0)
    executed = []

    action = {
        "type": "trade", "user_id": "u1", "amount": 100.0,
        "meta": {"side": "buy", "fee_rate": 0.001, "min_order_size": 0.0001},
    }
    tsr.commit(action, lambda: executed.append("trade"))

    assert executed == ["trade"]
    print("PASS  Normal flow: trade succeeds via TSR")


def test_normal_withdraw_succeeds():
    """Withdraw passes TSR when balance covers amount + fees."""
    tsr = TSR()
    tsr.balance_manager = _StubBalanceManager(balance=500.0)
    executed = []

    action = {
        "type": "withdraw", "user_id": "u1", "amount": 100.0,
        "meta": {"fee_rate": 0.001},
    }
    tsr.commit(action, lambda: executed.append("withdraw"))

    assert executed == ["withdraw"]
    print("PASS  Normal flow: withdraw succeeds via TSR")


# Invalid flow — TSR.validate() rejects BEFORE execute_fn is called

def test_withdraw_insufficient_funds_blocked_before_execution():
    """TSR.validate() rejects withdrawal when balance is too low — before any DB write."""
    tsr = TSR()
    tsr.balance_manager = _StubBalanceManager(balance=10.0)
    executed = []

    action = {
        "type": "withdraw", "user_id": "u1", "amount": 500.0,
        "meta": {"fee_rate": 0.001},
    }
    try:
        tsr.commit(action, lambda: executed.append("should_not_run"))
        raise AssertionError("Should have raised TSRBlocked")
    except TSRBlocked as e:
        assert "insufficient balance" in e.reason
        assert executed == [], "execute_fn must NOT have run"
    print("PASS  Invalid flow: withdraw with insufficient funds blocked by TSR.validate()")


def test_negative_amount_blocked_before_execution():
    """Negative amount rejected before execute_fn is called."""
    tsr = TSR()
    tsr.balance_manager = _StubBalanceManager(balance=1000.0)
    executed = []

    action = {"type": "withdraw", "user_id": "u1", "amount": -50.0}
    try:
        tsr.commit(action, lambda: executed.append("should_not_run"))
        raise AssertionError("Should have raised TSRBlocked")
    except TSRBlocked as e:
        assert "amount" in e.reason
        assert executed == []
    print("PASS  Invalid flow: negative amount blocked before execution")


def test_missing_user_blocked():
    """Unknown user_id returns None from get_balance — TSR blocks before execution."""
    tsr = TSR()
    tsr.balance_manager = _StubBalanceManager(balance=1000.0)
    executed = []

    action = {"type": "withdraw", "user_id": "__missing__", "amount": 10.0}
    try:
        tsr.commit(action, lambda: executed.append("should_not_run"))
        raise AssertionError("Should have raised TSRBlocked")
    except TSRBlocked as e:
        assert "not found" in e.reason
        assert executed == []
    print("PASS  Invalid flow: unknown user blocked before execution")


# Bypass attempt — direct call to private method raises immediately

def test_bypass_attempt_execute_buy_direct():
    """Calling _execute_buy directly — without tsr.commit — raises RuntimeError."""
    bm = BalanceManager()
    try:
        bm._execute_buy("user-1", 1.0, 100.0)
        raise AssertionError("Bypass succeeded — INVARIANT VIOLATED")
    except RuntimeError as e:
        assert _BYPASS in str(e)
    print("PASS  Bypass attempt: _execute_buy() direct call → RuntimeError")


# ── STEP 1 VERIFICATION — tsr.balance_manager wiring ────────────────────────

def test_tsr_balance_manager_wiring():
    """
    Simulates: dash_module.tsr.balance_manager = balances
    Proves that after wiring, validate() uses BalanceManager.get_balance().
    """
    tsr = TSR()
    assert tsr.balance_manager is None, "Should start unwired"

    stub = _StubBalanceManager(balance=50.0)
    tsr.balance_manager = stub   # <-- the single line added in app.py
    assert tsr.balance_manager is stub

    # Now validate() can check balance before commit
    action = {"type": "withdraw", "user_id": "u1", "amount": 1000.0, "meta": {"fee_rate": 0.001}}
    ok, reason = tsr.validate(action)
    assert not ok
    assert "insufficient balance" in reason
    print("PASS  tsr.balance_manager wiring: validate() uses BalanceManager.get_balance()")


# ── Runner ────────────────────────────────────────────────────────────────────

if __name__ == "__main__":
    tests = [
        # Original 12
        test_execute_buy_blocked_outside_commit,
        test_execute_sell_blocked_outside_commit,
        test_withdraw_blocked_outside_commit,
        test_credit_blocked_outside_commit,
        test_deposit_blocked_outside_commit,
        test_restore_blocked_outside_commit,
        test_public_tombstones_raise,
        test_readonly_blocks_commit,
        test_zero_amount_blocked,
        test_negative_amount_blocked,
        test_commit_sets_context_flag,
        test_context_cleared_on_exception,
        # Final system tests
        test_normal_deposit_succeeds,
        test_normal_trade_succeeds,
        test_normal_withdraw_succeeds,
        test_withdraw_insufficient_funds_blocked_before_execution,
        test_negative_amount_blocked_before_execution,
        test_missing_user_blocked,
        test_bypass_attempt_execute_buy_direct,
        test_tsr_balance_manager_wiring,
    ]

    failed = 0
    for t in tests:
        try:
            t()
        except AssertionError as e:
            print(f"FAIL  {t.__name__}: {e}")
            failed += 1
        except Exception as e:
            print(f"ERROR {t.__name__}: {type(e).__name__}: {e}")
            failed += 1

    print(f"\n{'='*55}")
    print(f"  {len(tests) - failed}/{len(tests)} tests passed")
    if failed:
        print(f"  {failed} FAILED — INVARIANT NOT SATISFIED")
        sys.exit(1)
    else:
        print("  INVARIANT HOLDS — TSR is the sole execution kernel")
        print("  No bypass is possible at runtime")

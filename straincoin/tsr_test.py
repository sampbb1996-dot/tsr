"""
TSR Runtime Invariant Tests
============================

Run with:  python -m pytest straincoin/tsr_test.py -v
Or:        python straincoin/tsr_test.py

These tests prove the invariant holds without a live database.
BalanceManager._private methods are tested directly — they must raise
RuntimeError("TSR_BYPASS_DETECTED") when called outside tsr.commit().
"""

import sys
import os
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

# ── Minimal stubs so we can import without a live DB ─────────────────────────

import types

# Stub out db module
db_stub = types.ModuleType("db")
db_stub.execute = lambda *a, **kw: []
db_stub.execute_in_tx = lambda *a, **kw: []
sys.modules["db"] = db_stub

from tsr import TSR, TSRBlocked, _tsr_ctx
from balance_manager import BalanceManager

_BYPASS = "TSR_BYPASS_DETECTED"


# ── Helper ────────────────────────────────────────────────────────────────────

def assert_bypass_blocked(fn, *args, **kwargs):
    """Assert that calling fn(*args) outside tsr.commit() raises RuntimeError."""
    try:
        fn(*args, **kwargs)
        raise AssertionError(f"{fn.__name__} should have raised RuntimeError but did not")
    except RuntimeError as e:
        assert _BYPASS in str(e), f"Wrong error: {e}"


def assert_raises(exc_type, fn, *args, **kwargs):
    try:
        fn(*args, **kwargs)
        raise AssertionError(f"Expected {exc_type.__name__} but no exception was raised")
    except exc_type:
        pass


# ── STEP 5: Runtime bypass tests ──────────────────────────────────────────────

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


# ── Public tombstone tests ────────────────────────────────────────────────────

def test_public_tombstones_raise():
    bm = BalanceManager()
    for method_name in ("credit_confirmed_deposit", "deposit", "withdraw",
                        "restore_withdrawal", "execute_buy", "execute_sell"):
        fn = getattr(bm, method_name)
        assert_raises(RuntimeError, fn)
        print(f"PASS  {method_name}() raises RuntimeError (tombstone)")


# ── TSR regime tests ──────────────────────────────────────────────────────────

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
    """Prove that _tsr_ctx.active is True inside execute_fn and False outside."""
    tsr = TSR()
    seen_inside = []

    def probe():
        seen_inside.append(getattr(_tsr_ctx, "active", False))
        return "ok"

    action = {"type": "trade", "user_id": "u1", "amount": 1.0}
    tsr.commit(action, probe)

    assert seen_inside == [True], f"Expected [True], got {seen_inside}"
    assert not getattr(_tsr_ctx, "active", False), "_tsr_ctx.active should be False after commit"
    print("PASS  _tsr_ctx.active=True inside commit(), False after")


def test_context_cleared_on_exception():
    """Prove that _tsr_ctx.active is cleared even if execute_fn raises."""
    tsr = TSR()

    def boom():
        raise ValueError("simulated failure")

    action = {"type": "trade", "user_id": "u1", "amount": 1.0}
    try:
        tsr.commit(action, boom)
    except ValueError:
        pass

    assert not getattr(_tsr_ctx, "active", False), \
        "_tsr_ctx.active should be False after execute_fn exception"
    print("PASS  _tsr_ctx.active cleared even when execute_fn raises")


# ── Runner ────────────────────────────────────────────────────────────────────

if __name__ == "__main__":
    tests = [
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

    print(f"\n{'='*50}")
    print(f"  {len(tests) - failed}/{len(tests)} tests passed")
    if failed:
        print(f"  {failed} FAILED")
        sys.exit(1)
    else:
        print("  INVARIANT HOLDS — TSR bypass is structurally impossible")

"""
Tests for bias.py — Structural Bias Detector
Run: python straincoin/bias_test.py
"""

import sys, os
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from bias import compute_bias, compute_gate

PASS = 0
FAIL = 0

def check(label, got, expected, tol=1e-9):
    global PASS, FAIL
    if abs(got - expected) <= tol:
        print(f"PASS  {label}")
        PASS += 1
    else:
        print(f"FAIL  {label}  got={got!r}  expected={expected!r}")
        FAIL += 1

def check_eq(label, got, expected):
    global PASS, FAIL
    if got == expected:
        print(f"PASS  {label}")
        PASS += 1
    else:
        print(f"FAIL  {label}  got={got!r}  expected={expected!r}")
        FAIL += 1

# ── compute_bias ────────────────────────────────────────────────────────────

# B1: balanced activity → low spread
r = compute_bias([10.0, 10.0, 10.0], 100.0, 100.0, {"A": 1, "B": 1})
check("B1=0 for uniform activity", r["components"]["activity_imbalance"], 0.0)
check("B2=0 for equal volumes", r["components"]["directional_imbalance"], 0.0)
check("B3=0.5 for two equal primitives", r["components"]["primitive_concentration"], 0.5)
check("bias_score = mean(0, 0, 0.5)", r["bias_score"], 0.5/3, tol=1e-5)

# B2: full directional imbalance
r = compute_bias([1.0], 100.0, 0.0, {"X": 1})
check("B2=1.0 when sell=0", r["components"]["directional_imbalance"], 1.0)

# B3: single dominant primitive
r = compute_bias([1.0], 50.0, 50.0, {"only": 99})
check("B3=1.0 for single primitive", r["components"]["primitive_concentration"], 1.0)

# Zero-safe: empty primitives
r = compute_bias([1.0], 0.0, 0.0, {})
check("B2=0 when both volumes=0", r["components"]["directional_imbalance"], 0.0)
check("B3=0 when no primitives", r["components"]["primitive_concentration"], 0.0)

# Zero-safe: single activity value (stdev undefined → 0)
r = compute_bias([5.0], 10.0, 20.0, {"A": 3, "B": 1})
check("B1=0 for single-element activity", r["components"]["activity_imbalance"], 0.0)

# ── compute_gate ────────────────────────────────────────────────────────────

check_eq("gate=1.0 when bias=0.0",   compute_gate(0.0),    1.0)
check_eq("gate=1.0 when bias=0.299", compute_gate(0.299),  1.0)
check_eq("gate=0.5 when bias=0.3",   compute_gate(0.3),    0.5)
check_eq("gate=0.5 when bias=0.599", compute_gate(0.599),  0.5)
check_eq("gate=0.1 when bias=0.6",   compute_gate(0.6),    0.1)
check_eq("gate=0.1 when bias=1.0",   compute_gate(1.0),    0.1)

# ── gate scales contribution and confidence correctly ───────────────────────

bias_low  = compute_bias([1.0, 1.0], 50.0, 50.0, {"A": 1, "B": 1})   # score ≈ 0.167
bias_high = compute_bias([1.0, 100.0], 90.0, 10.0, {"D": 99, "E": 1}) # score > 0.6

gate_low  = compute_gate(bias_low["bias_score"])
gate_high = compute_gate(bias_high["bias_score"])

contribution = 100.0
check_eq("low-bias gate=1.0 → contribution unchanged", contribution * gate_low,  100.0)
check_eq("high-bias gate=0.1 → contribution scaled",   contribution * gate_high, 10.0)

# ── summary ─────────────────────────────────────────────────────────────────

print()
print("=" * 55)
print(f"  {PASS + FAIL} tests  —  {PASS} passed  {FAIL} failed")
if FAIL == 0:
    print("  STRUCTURAL BIAS DETECTOR VERIFIED")
print("=" * 55)

"""
Tests for bias.py — Structural Bias Detector
Run: python straincoin/bias_test.py
"""

import sys, os
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from bias import compute_bias, compute_gate, BiasHistory, compute_persistent_gate

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

# ── BiasHistory ─────────────────────────────────────────────────────────────

h = BiasHistory(N=5)
check_eq("empty history → persistent_bias=0.0", h.persistent_bias, 0.0)
check_eq("empty history → not full", h.full, False)
check_eq("empty history → len=0", len(h), 0)

h.append(0.2)
h.append(0.4)
check("two scores → persistent_bias=0.3", h.persistent_bias, 0.3, tol=1e-9)
check_eq("two scores → not full (N=5)", h.full, False)

for _ in range(3):
    h.append(0.8)   # push window to [0.2, 0.4, 0.8, 0.8, 0.8]
check_eq("five scores → full", h.full, True)
check("full window mean", h.persistent_bias, (0.2+0.4+0.8+0.8+0.8)/5, tol=1e-9)

h.append(0.1)       # oldest (0.2) evicted → [0.4, 0.8, 0.8, 0.8, 0.1]
check("oldest score evicted (maxlen enforced)", h.persistent_bias, (0.4+0.8+0.8+0.8+0.1)/5, tol=1e-9)

# constructor guard
try:
    BiasHistory(N=0)
    print("FAIL  BiasHistory(N=0) should raise ValueError")
    FAIL += 1
except ValueError:
    print("PASS  BiasHistory(N=0) raises ValueError")
    PASS += 1

# ── compute_persistent_gate ──────────────────────────────────────────────────

# Below 0.6 → delegates to compute_gate thresholds
check_eq("persistent_gate: 0.0 → 1.0",   compute_persistent_gate(0.0),   1.0)
check_eq("persistent_gate: 0.299 → 1.0",  compute_persistent_gate(0.299), 1.0)
check_eq("persistent_gate: 0.3 → 0.5",   compute_persistent_gate(0.3),   0.5)
check_eq("persistent_gate: 0.599 → 0.5",  compute_persistent_gate(0.599), 0.5)
check_eq("persistent_gate: 0.6 → 0.1",   compute_persistent_gate(0.6),   0.1)

# Above 0.6 → freeze (0.0)
check_eq("persistent_gate: 0.601 → 0.0 (freeze)", compute_persistent_gate(0.601), 0.0)
check_eq("persistent_gate: 1.0 → 0.0 (freeze)",   compute_persistent_gate(1.0),   0.0)

# ── end-to-end: spike vs recovery ───────────────────────────────────────────

history = BiasHistory(N=10)

# Phase 1: single high-bias reading — both instant and persistent gates fire
spike_score = 0.9
instant_gate = compute_gate(spike_score)
history.append(spike_score)
persistent_gate_after_spike = compute_persistent_gate(history.persistent_bias)
check_eq("spike: instant gate=0.1",                  instant_gate,                 0.1)
check_eq("spike: persistent gate=0.0 (mean=0.9>0.6)", persistent_gate_after_spike, 0.0)

# Phase 2: sustained high bias — still frozen (all 10 readings at 0.9)
for _ in range(9):
    history.append(0.9)
check_eq("sustained: persistent_bias > 0.6 → gate=0.0 (freeze)",
         compute_persistent_gate(history.persistent_bias), 0.0)

# Phase 3: bias recovers — low scores displace the old high ones
history2 = BiasHistory(N=5)
for _ in range(5):
    history2.append(0.1)   # five low readings, mean=0.1
check_eq("recovery: persistent_bias=0.1 → gate=1.0",
         compute_persistent_gate(history2.persistent_bias), 1.0)

# ── summary ─────────────────────────────────────────────────────────────────

print()
print("=" * 55)
print(f"  {PASS + FAIL} tests  —  {PASS} passed  {FAIL} failed")
if FAIL == 0:
    print("  STRUCTURAL BIAS DETECTOR VERIFIED")
    print("  PERSISTENT BIAS TRACKER VERIFIED")
print("=" * 55)

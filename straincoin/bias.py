"""
Structural Bias Detector — PRIM
================================

Evaluates input imbalance BEFORE primitive extraction or pattern logic runs.
Pure functions only (except BiasHistory which holds a rolling window).
No dependencies beyond stdlib.

compute_bias()            — returns bias_score and components
compute_gate()            — returns gate multiplier from a single bias_score
BiasHistory               — rolling window of N bias scores
compute_persistent_gate() — gate based on sustained (rolling-mean) bias
"""

import statistics
from collections import deque


def compute_bias(
    activity: list[float],
    buy_volume: float,
    sell_volume: float,
    primitives: dict[str, int],
) -> dict:
    """
    Compute structural bias across three dimensions.

    Parameters
    ──────────
    activity    — list of activity values (per source / agent / window)
    buy_volume  — total buy volume (float)
    sell_volume — total sell volume (float)
    primitives  — {primitive_name: count}

    Returns
    ───────
    {
        "bias_score": float,          # mean of B1, B2, B3  (0.0 – 1.0)
        "components": {
            "activity_imbalance":      float,  # B1
            "directional_imbalance":   float,  # B2
            "primitive_concentration": float,  # B3
        }
    }
    """
    # B1 — activity spread relative to mean
    if len(activity) >= 2:
        mean_a = statistics.mean(activity)
        B1 = (statistics.stdev(activity) / mean_a) if mean_a != 0 else 0.0
    else:
        B1 = 0.0

    # B2 — directional imbalance between buy and sell
    total_volume = buy_volume + sell_volume
    B2 = abs(buy_volume - sell_volume) / total_volume if total_volume != 0 else 0.0

    # B3 — concentration of the dominant primitive
    prim_total = sum(primitives.values())
    B3 = max(primitives.values()) / prim_total if prim_total != 0 and primitives else 0.0

    bias_score = (B1 + B2 + B3) / 3

    return {
        "bias_score": round(bias_score, 6),
        "components": {
            "activity_imbalance":      round(B1, 6),
            "directional_imbalance":   round(B2, 6),
            "primitive_concentration": round(B3, 6),
        },
    }


def compute_gate(bias_score: float) -> float:
    """
    Convert a bias score into a gate multiplier.

    bias < 0.3          → 1.0  (full pass)
    0.3 ≤ bias < 0.6    → 0.5  (half weight)
    bias ≥ 0.6          → 0.1  (near-block)
    """
    if bias_score < 0.3:
        return 1.0
    if bias_score < 0.6:
        return 0.5
    return 0.1


class BiasHistory:
    """
    Rolling window of bias scores.

    Tracks the last N bias_score values so callers can detect sustained
    distortion rather than reacting to isolated spikes.

    Usage
    ─────
        history = BiasHistory(N=20)          # module-level singleton
        history.append(bias_data["bias_score"])
        gate = compute_persistent_gate(history.persistent_bias)
    """

    def __init__(self, N: int = 20):
        if N < 1:
            raise ValueError("N must be >= 1")
        self._window: deque[float] = deque(maxlen=N)
        self.N = N

    def append(self, bias_score: float) -> None:
        """Add the latest bias_score to the rolling window."""
        self._window.append(bias_score)

    @property
    def persistent_bias(self) -> float:
        """
        Rolling mean of all scores collected so far.
        Returns 0.0 when the window is empty.
        """
        if not self._window:
            return 0.0
        return sum(self._window) / len(self._window)

    @property
    def full(self) -> bool:
        """True once N scores have been collected."""
        return len(self._window) == self.N

    def __len__(self) -> int:
        return len(self._window)


def compute_persistent_gate(persistent_bias: float) -> float:
    """
    Gate multiplier based on *sustained* bias (rolling mean).

    A single high-bias tick is handled by compute_gate(); this function
    acts only when distortion has persisted across multiple windows.

    persistent_bias ≤ 0.6  → same thresholds as compute_gate()
    persistent_bias > 0.6  → 0.0  (freeze — learning fully suspended)

    The freeze threshold matches the screenshot spec:
        if persistent_bias > 0.6:
            # freeze or heavily damp learning
    """
    if persistent_bias > 0.6:
        return 0.0          # freeze
    return compute_gate(persistent_bias)

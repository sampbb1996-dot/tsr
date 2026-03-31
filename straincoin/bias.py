"""
Structural Bias Detector — PRIM
================================

Evaluates input imbalance BEFORE primitive extraction or pattern logic runs.
Pure functions only. No side effects. No dependencies beyond stdlib.

compute_bias()  — returns bias_score and components
compute_gate()  — returns gate multiplier from bias_score
"""

import statistics


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

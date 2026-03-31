"""
Bias Integration Points — PRIM
================================

Copy these snippets into the relevant sections of dashboard.py.
They show ONLY the changed/added lines — do not rewrite surrounding code.

────────────────────────────────────────────────────────────────────────────
IMPORT (top of dashboard.py, with other imports)
────────────────────────────────────────────────────────────────────────────

    from bias import compute_bias, compute_gate

────────────────────────────────────────────────────────────────────────────
SHARED HELPER — call once per request where bias is needed
────────────────────────────────────────────────────────────────────────────

Add this function anywhere in dashboard.py (near the other helpers):

    def _current_bias() -> dict:
        \"\"\"
        Compute the current structural bias from live engine state.
        Returns compute_bias() result, or a zero-bias default if engines
        are not yet initialised.
        \"\"\"
        # Activity values — one float per agent/source
        activity = []
        if agent_manager is not None:
            try:
                activity = [float(a.get("activity", 0))
                            for a in agent_manager.get_agent_states()
                            if a.get("activity") is not None]
            except Exception:
                pass
        if not activity:
            activity = [1.0]    # neutral — no imbalance

        # Buy / sell volumes from exchange
        buy_vol, sell_vol = 0.0, 0.0
        if exchange is not None:
            try:
                buy_vol  = float(exchange.get_buy_volume()  or 0)
                sell_vol = float(exchange.get_sell_volume() or 0)
            except Exception:
                pass

        # Primitive counts from discovery layer
        prim_counts: dict[str, int] = {}
        if prim_disc is not None:
            try:
                prim_counts = dict(prim_disc.get_counts() or {})
            except Exception:
                pass
        if not prim_counts:
            prim_counts = {"_none": 1}  # neutral — single primitive

        return compute_bias(activity, buy_vol, sell_vol, prim_counts)

────────────────────────────────────────────────────────────────────────────
INTEGRATION POINT 1 — /api/action  (contribution_score scaling)
────────────────────────────────────────────────────────────────────────────

FIND in api_action() (or the equivalent route that produces contribution_score):

    contribution_score = ...   # however it is currently computed

ADD immediately after:

    bias_data = _current_bias()
    gate      = compute_gate(bias_data["bias_score"])
    contribution_score = round(contribution_score * gate, 6)

────────────────────────────────────────────────────────────────────────────
INTEGRATION POINT 2 — /api/prediction  (confidence scaling)
────────────────────────────────────────────────────────────────────────────

FIND in api_prediction() where prediction confidence is returned, e.g.:

    confidence = some_value

ADD immediately after:

    bias_data  = _current_bias()
    gate       = compute_gate(bias_data["bias_score"])
    confidence = round(confidence * gate, 6)

────────────────────────────────────────────────────────────────────────────
NEW ROUTE — GET /api/bias
────────────────────────────────────────────────────────────────────────────

Add this route to dashboard.py:

    @app.route("/api/bias")
    def api_bias():
        bias_data = _current_bias()
        gate      = compute_gate(bias_data["bias_score"])
        return jsonify({
            "bias_score": bias_data["bias_score"],
            "gate":       gate,
            "components": bias_data["components"],
        })
"""

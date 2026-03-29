"""
TSR INTEGRATION PATCH — dashboard.py
======================================

Apply the four changes below to dashboard.py.
Each section is marked with the line(s) to find and what to insert/replace.

────────────────────────────────────────────────────────────────────────────
CHANGE 1 — Import and initialise TSR (top of dashboard.py, near other imports)
────────────────────────────────────────────────────────────────────────────

ADD after existing imports:

    from tsr import TSR
    tsr = TSR()

────────────────────────────────────────────────────────────────────────────
CHANGE 2 — Wrap trade execution
────────────────────────────────────────────────────────────────────────────

FIND any block that calls execute_trade() (or equivalent), e.g.:

    result = execute_trade(user_id, amount, ...)

REPLACE with:

    action = {
        "type":    "trade",
        "user_id": user_id,
        "amount":  amount,
        "meta":    {"price": price, "side": side},
    }
    try:
        result = tsr.commit(action, lambda: execute_trade(user_id, amount, ...))
    except Exception as e:
        if str(e) == "TSR_BLOCKED":
            return jsonify({"error": "blocked_by_tsr"}), 403
        raise

────────────────────────────────────────────────────────────────────────────
CHANGE 3 — Wrap withdrawal execution
────────────────────────────────────────────────────────────────────────────

FIND the withdrawal execution call, e.g.:

    result = real_withdraw(user_id, amount)

REPLACE with:

    action = {
        "type":    "withdraw",
        "user_id": user_id,
        "amount":  amount,
    }
    try:
        result = tsr.commit(action, lambda: real_withdraw(user_id, amount))
    except Exception as e:
        if str(e) == "TSR_BLOCKED":
            return jsonify({"error": "blocked_by_tsr"}), 403
        raise

────────────────────────────────────────────────────────────────────────────
CHANGE 4 — Wrap balance credit inside Stripe webhook ONLY
────────────────────────────────────────────────────────────────────────────

DO NOT wrap deposit initiation (the /api/deposit/initiate route).
ONLY wrap the credit call inside the Stripe webhook handler.

FIND inside the webhook handler (the route that receives Stripe events):

    balance_manager.credit_confirmed_deposit(user_id, amount, deposit_id, stripe_pi_id)

REPLACE with:

    action = {
        "type":    "deposit",
        "user_id": user_id,
        "amount":  amount,
        "meta":    {"deposit_id": deposit_id, "stripe_pi_id": stripe_pi_id},
    }
    try:
        tsr.commit(action, lambda: balance_manager.credit_confirmed_deposit(
            user_id, amount, deposit_id, stripe_pi_id
        ))
    except Exception as e:
        if str(e) == "TSR_BLOCKED":
            app.logger.error(f"[TSR] deposit credit blocked for user={user_id}")
            return jsonify({"error": "blocked_by_tsr"}), 403
        raise

────────────────────────────────────────────────────────────────────────────
CHANGE 5 — Regime control endpoint (add as a new Flask route)
────────────────────────────────────────────────────────────────────────────

ADD this route anywhere in dashboard.py with the other API routes:

    @app.route("/api/tsr/regime", methods=["POST"])
    def api_tsr_regime():
        body   = request.get_json(force=True) or {}
        regime = body.get("regime", "").upper()
        if regime not in ("CALM", "READONLY"):
            return jsonify({"error": "regime must be CALM or READONLY"}), 400
        tsr.set_regime(regime)
        return jsonify({"regime": tsr.get_regime()})

    @app.route("/api/tsr/regime", methods=["GET"])
    def api_tsr_regime_get():
        return jsonify({"regime": tsr.get_regime()})

────────────────────────────────────────────────────────────────────────────
CHANGE 6 — Inject tsr into dashboard from app.py (if using singletons)
────────────────────────────────────────────────────────────────────────────

In app.py _init_engines(), after the other dash_module.xxx = yyy lines, add:

    # tsr is already initialised at module level in dashboard.py — no injection needed.
    # To switch regime at boot based on env:
    import os
    startup_regime = os.environ.get("TSR_REGIME", "CALM").upper()
    dash_module.tsr.set_regime(startup_regime)
"""

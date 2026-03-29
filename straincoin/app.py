"""
StrainCoin / PRIM — Main Entry Point
=====================================

Bootstraps all components in the correct order, starts background threads,
then serves the Flask dashboard.

Launch modes
────────────
  python straincoin/app.py               # simulation (synthetic agents)
  python straincoin/app.py --onchain     # live PRIM on-chain data feed

  --onchain requires these env vars (set in .env or Replit Secrets):
    PRIM_RPC_URL          — Ethereum-compatible HTTP RPC endpoint
    PRIM_PAIR_ADDRESS     — Uniswap V2 PRIM/WETH pair address
    PRIM_CONTRACT_ADDRESS — PRIM ERC-20 contract address (optional but helpful)

  If PRIM_RPC_URL is absent or the RPC is unreachable the system
  automatically falls back to simulation mode so the engine keeps running.
"""

import argparse
import os
import sys
import time
import threading

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from orderbook            import Exchange
from agents               import AgentManager
from strain_engine        import StrainEngine
from primitive_discovery  import PrimitiveDiscovery
from sequence_engine      import SequenceEngine
from onchain_feed         import OnchainFeed
from issuance_tracker     import IssuanceTracker
from balance_manager      import BalanceManager
from contribution_engine  import StructureResolver
import dashboard as dash_module


def _stream_printer(prim_disc, interval: float = 10.0):
    """Prints the PRIM language stream to stdout."""
    last_printed = 0
    while True:
        time.sleep(1)
        stream = prim_disc.get_stream(100)
        if not stream:
            continue
        latest = stream[-1]
        ts = latest['timestamp']
        if ts > last_printed:
            label = time.strftime('%H:%M:%S', time.localtime(ts))
            print(f'{label}  {latest["primitive"]}', flush=True)
            last_printed = ts


def _init_engines(onchain: bool = False):
    """
    Initialise all engine components in a background thread so Flask can
    start accepting connections immediately (avoids the blank-screen cold-start
    delay on autoscale deployments).
    """
    print('[1/7] Starting exchange …', flush=True)
    exchange = Exchange()

    print('[2/7] Agent manager initialised (mixed mode — simulation + user activity) …', flush=True)
    agents = AgentManager(exchange, mode='mixed')
    agents.start()

    onchain_feed = None
    if onchain:
        print('[3/7] Starting on-chain feed …', flush=True)
        onchain_feed = OnchainFeed(exchange)
        live = onchain_feed.start()
        if live:
            print('      Connected to on-chain PRIM data.', flush=True)
        else:
            print('      Could not connect — falling back to simulation.', flush=True)
    else:
        print('[3/7] On-chain feed skipped (simulation mode).', flush=True)

    print('[4/7] Starting strain engine …', flush=True)
    engine = StrainEngine(exchange)
    print('[4/7]  → Warming up from historical trades …', flush=True)
    engine.warm_up_from_csv()
    engine.start()

    print('[5/7] Starting primitive discovery …', flush=True)
    prim = PrimitiveDiscovery(engine)
    prim.start()

    print('[6/7] Starting sequence engine …', flush=True)
    seq = SequenceEngine(prim)
    seq.start()

    print('[+] Starting issuance tracker (activity → PRIM supply) …', flush=True)
    issuance = IssuanceTracker(exchange, engine)
    issuance.start()

    print('[+] Initialising balance manager (real-stakes user accounts) …', flush=True)
    balances = BalanceManager()

    print('[+] Starting structure resolver (temporal prediction/outcome separation) …', flush=True)
    def _get_future_strain():
        w, _ = engine.get_current()
        if w is None:
            return {}
        return {
            'Sd': float(w.Sd),
            'Sv': float(w.Sv),
            'Sa': float(w.Sa),
            'Sp': float(w.Sp),
            'Sr': float(w.Sr),
        }
    resolver = StructureResolver(_get_future_strain)
    resolver.start()

    threading.Thread(
        target=_stream_printer, args=(prim,),
        daemon=True, name='stream-printer',
    ).start()

    # Inject singletons into dashboard — Flask endpoints read these at
    # request time so setting them now (after Flask is already running)
    # is safe.
    dash_module.exchange         = exchange
    dash_module.agent_manager    = agents
    dash_module.strain_engine    = engine
    dash_module.prim_disc        = prim
    dash_module.seq_engine       = seq
    dash_module.onchain_feed     = onchain_feed
    dash_module.issuance_tracker = issuance
    dash_module.balance_manager  = balances
    dash_module.tsr.balance_manager = balances  # wire TSR validation to live BalanceManager

    import language_activity as _la
    _la.init(prim, seq)

    dash_module.ensure_guest_schema()

    print('PRIM SYSTEM — engines ready.', flush=True)
    print('-' * 40, flush=True)


def run(onchain: bool = False):
    print('=' * 55, flush=True)
    print('  PRIM — Activity-Driven Symbolic Language Engine', flush=True)
    print('=' * 55, flush=True)

    mode_label = 'on-chain (PRIM/WETH)' if onchain else 'simulation'
    print(f'  Mode: {mode_label}\n', flush=True)

    # Start engines in a background thread so Flask is available immediately.
    # Dashboard endpoints return {"available": false} while engines are None,
    # preventing a blank-screen cold-start delay.
    init_thread = threading.Thread(
        target=_init_engines, args=(onchain,),
        daemon=True, name='engine-init',
    )
    init_thread.start()

    port = int(os.environ.get('PORT', 5000))
    print(f'\n  Dashboard → http://0.0.0.0:{port}', flush=True)
    print('  Engines initialising in background …\n', flush=True)

    dash_module.app.run(host='0.0.0.0', port=port, debug=False, threaded=True)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(
        description='PRIM — Activity-Driven Symbolic Language Engine'
    )
    parser.add_argument(
        '--onchain',
        action='store_true',
        help='Feed from real PRIM on-chain trading data instead of simulation agents',
    )
    args = parser.parse_args()
    run(onchain=args.onchain)

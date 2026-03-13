import type { Status, Cost } from "../api";

interface StatsGridProps {
  status: Status | null;
  cost: Cost | null;
}

export function StatsGrid({ status, cost }: StatsGridProps) {
  const walletValue = status?.wallet_value ?? 0;
  const todayPnl = status?.today_pnl ?? 0;
  const dryRun = status?.dry_run ?? true;

  return (
    <section className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-6">
      <div className="glass rounded-lg p-4">
        <p className="text-[9px] text-cyan-600 uppercase tracking-wider">
          Wallet Value
        </p>
        <p className="text-2xl font-bold font-orbitron text-cyan-300">
          {walletValue > 0 ? `$${walletValue.toFixed(2)}` : "$0.00"}
        </p>
        <p className="text-[9px] text-cyan-700 mt-1">
          {dryRun ? "Credits (dry run)" : "Credits"}
        </p>
      </div>
      <div className="glass rounded-lg p-4">
        <p className="text-[9px] text-cyan-600 uppercase tracking-wider">
          P&L Today
        </p>
        <p
          className={`text-2xl font-bold font-orbitron ${
            todayPnl >= 0 ? "text-green-400" : "text-red-400"
          }`}
        >
          ${todayPnl.toFixed(2)}
        </p>
      </div>
      <div className="glass rounded-lg p-4">
        <p className="text-[9px] text-cyan-600 uppercase tracking-wider">
          Risk Level
        </p>
        <p className="text-xl font-bold font-orbitron text-green-400">LOW</p>
      </div>
    </section>
  );
}

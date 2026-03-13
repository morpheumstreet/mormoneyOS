import type { Status } from "../api";

interface HeaderProps {
  status: Status | null;
}

export function Header({ status }: HeaderProps) {
  const isRunning = status?.is_running ?? false;
  const tickCount = status?.tick_count ?? 0;

  return (
    <header className="flex flex-wrap items-center justify-between gap-4 mb-6">
      <div>
        <h1 className="font-orbitron text-2xl md:text-3xl font-bold text-cyan-400">
          MONEYCLAW v{status?.version ?? "0.1.0"}
        </h1>
        <p className="text-cyan-600 text-sm mt-1">Sovereign AI Agent Runtime</p>
      </div>
      <div className="flex items-center gap-3">
        <span
          className={`px-3 py-1 rounded text-xs font-mono border ${
            isRunning
              ? "bg-green-500/20 text-green-400 border-green-500/30"
              : "bg-yellow-500/20 text-yellow-400 border-yellow-500/30"
          }`}
        >
          {isRunning ? "SYSTEM ONLINE" : "PAUSED"}
        </span>
        <span className="text-cyan-600">|</span>
        <span className="font-orbitron text-sm">Tick #{tickCount}</span>
      </div>
    </header>
  );
}

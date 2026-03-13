import type { Strategy } from "../api";

interface StrategiesListProps {
  strategies: Strategy[];
  onSelect: (s: Strategy) => void;
}

export function StrategiesList({ strategies, onSelect }: StrategiesListProps) {
  return (
    <section className="glass rounded-lg p-4">
      <h2 className="font-orbitron text-lg text-cyan-400 mb-4">
        ACTIVE STRATEGIES
      </h2>
      <div className="space-y-2">
        {!strategies?.length ? (
          <p className="text-cyan-600 text-sm">No strategies or skills loaded.</p>
        ) : (
          strategies.map((s, i) => (
            <div
              key={`${s.name}-${i}`}
              className="strategy-item rounded p-3 border border-cyan-600/20"
              onClick={() => onSelect(s)}
              onKeyDown={(e) => e.key === "Enter" && onSelect(s)}
              role="button"
              tabIndex={0}
            >
              <p className="font-orbitron text-cyan-400">{s.name}</p>
              <p className="text-cyan-600 text-sm mt-1">{s.description || ""}</p>
              <div className="flex gap-2 mt-2">
                <span
                  className={`text-[9px] ${
                    s.enabled ? "text-green-400" : "text-gray-500"
                  }`}
                >
                  {s.enabled ? "● ACTIVE" : "○ OFFLINE"}
                </span>
                <span className="text-[9px] text-cyan-600">
                  {(s.risk_level || "low").toUpperCase()}
                </span>
              </div>
            </div>
          ))
        )}
      </div>
    </section>
  );
}

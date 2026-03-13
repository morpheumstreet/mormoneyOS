import { useEffect } from "react";
import type { Strategy } from "../api";

interface StrategyModalProps {
  strategy: Strategy | null;
  onClose: () => void;
}

export function StrategyModal({ strategy, onClose }: StrategyModalProps) {
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [onClose]);

  if (!strategy) return null;

  return (
    <div
      className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 p-4"
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div className="glass rounded-lg max-w-lg w-full max-h-[90vh] overflow-y-auto p-6">
        <div className="flex justify-between items-start mb-4">
          <h3 className="font-orbitron text-xl text-cyan-400">{strategy.name}</h3>
          <button
            onClick={onClose}
            className="text-cyan-600 hover:text-cyan-400"
            aria-label="Close"
          >
            ✕
          </button>
        </div>
        <p className="text-cyan-300 text-sm mb-4">
          {strategy.description || "No description."}
        </p>
        <div className="flex gap-2 mb-4">
          <span
            className={`px-2 py-1 text-xs font-mono rounded ${
              strategy.enabled ? "bg-green-500/20 text-green-400" : "bg-gray-500/20 text-gray-400"
            }`}
          >
            {strategy.enabled ? "ACTIVE" : "OFFLINE"}
          </span>
          <span className="px-2 py-1 text-xs font-mono rounded bg-cyan-500/20 text-cyan-400">
            RISK: {(strategy.risk_level || "low").toUpperCase()}
          </span>
        </div>
        <div className="grid grid-cols-2 gap-4 text-sm mb-4">
          <div>
            <p className="text-cyan-600 text-[9px] uppercase">ROI Estimate</p>
            <p className="font-orbitron text-cyan-300">N/A</p>
          </div>
          <div>
            <p className="text-cyan-600 text-[9px] uppercase">Total Trades</p>
            <p className="font-orbitron text-cyan-300">0</p>
          </div>
        </div>
        <button
          onClick={onClose}
          className="w-full py-2 border border-cyan-500/50 rounded font-orbitron text-sm hover:bg-cyan-600/20 transition"
        >
          CLOSE
        </button>
      </div>
    </div>
  );
}

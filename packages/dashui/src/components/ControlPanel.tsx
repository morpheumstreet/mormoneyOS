interface ControlPanelProps {
  onPause: () => Promise<void>;
  onResume: () => Promise<void>;
}

export function ControlPanel({ onPause, onResume }: ControlPanelProps) {
  return (
    <section className="glass rounded-lg p-4">
      <h2 className="font-orbitron text-lg text-cyan-400 mb-4">CONTROL</h2>
      <div className="flex gap-2">
        <button
          onClick={onPause}
          className="flex-1 px-4 py-2 bg-yellow-600/20 border border-yellow-500/50 rounded font-orbitron text-sm hover:bg-yellow-600/30 transition"
        >
          PAUSE
        </button>
        <button
          onClick={onResume}
          className="flex-1 px-4 py-2 bg-green-600/20 border border-green-500/50 rounded font-orbitron text-sm hover:bg-green-600/30 transition"
        >
          RESUME
        </button>
      </div>
    </section>
  );
}

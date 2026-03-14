import { useEffect, useState } from "react";
import { Wrench, AlertTriangle, Loader2 } from "lucide-react";
import { useWalletAuth } from "@/contexts/WalletAuthContext";
import { getTools, patchToolEnabled, type ToolItem } from "@/lib/api";

export default function ConfigTools() {
  const { hasWriteAccess } = useWalletAuth();
  const [tools, setTools] = useState<ToolItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [toggling, setToggling] = useState<Record<string, boolean>>({});

  useEffect(() => {
    getTools()
      .then((res) => setTools(res.tools || []))
      .catch((e) => setError(e instanceof Error ? e.message : "Load failed"))
      .finally(() => setLoading(false));
  }, []);

  const handleToggle = async (tool: ToolItem) => {
    if (!hasWriteAccess) return;
    const nextEnabled = !tool.enabled;
    setToggling((prev) => ({ ...prev, [tool.name]: true }));
    try {
      await patchToolEnabled(tool.name, nextEnabled);
      setTools((prev) =>
        prev.map((t) =>
          t.name === tool.name ? { ...t, enabled: nextEnabled } : t
        )
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : "Update failed");
    } finally {
      setToggling((prev) => ({ ...prev, [tool.name]: false }));
    }
  };

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="electric-loader h-12 w-12 rounded-full" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <div className="electric-icon h-10 w-10 rounded-xl flex items-center justify-center">
          <Wrench className="h-5 w-5 text-[#9bc3ff]" />
        </div>
        <div>
          <h2 className="text-lg font-semibold text-white">Tools</h2>
          <p className="text-sm text-[#8aa8df]">
            Enable or disable agent tools. Disabled tools are hidden from the LLM.
          </p>
        </div>
      </div>

      {!hasWriteAccess && (
        <div className="electric-card p-4 border-amber-500/30 bg-amber-950/20">
          <div className="flex items-start gap-3">
            <AlertTriangle className="h-5 w-5 text-amber-400 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-amber-200">
                Write access required
              </p>
              <p className="text-sm text-amber-300/80 mt-1">
                Connect your wallet and sign to toggle tools.
              </p>
            </div>
          </div>
        </div>
      )}

      {error && (
        <div className="electric-card p-3 border-rose-500/30 bg-rose-950/20 flex items-center gap-2">
          <AlertTriangle className="h-4 w-4 text-rose-400 flex-shrink-0" />
          <span className="text-sm text-rose-300">{error}</span>
        </div>
      )}

      <div className="electric-card overflow-hidden">
        <div className="divide-y divide-[#1a3670]">
          {tools.length === 0 ? (
            <div className="px-4 py-8 text-center text-sm text-[#8aa8df]">
              No tools available. Ensure the agent is running with tools configured.
            </div>
          ) : (
            tools.map((tool) => (
              <div
                key={tool.name}
                className="flex items-center justify-between gap-4 px-4 py-3 hover:bg-[#07132f]/50 transition-colors"
              >
                <div className="min-w-0 flex-1">
                  <p className="font-medium text-white">{tool.name}</p>
                  {tool.description && (
                    <p className="mt-0.5 text-xs text-[#8aa8df] line-clamp-2">
                      {tool.description}
                    </p>
                  )}
                </div>
                <button
                  type="button"
                  onClick={() => handleToggle(tool)}
                  disabled={!hasWriteAccess || !!toggling[tool.name]}
                  className={`
                    relative inline-flex h-7 w-12 shrink-0 items-center rounded-full
                    transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-[#4f83ff] focus:ring-offset-2 focus:ring-offset-[#050d1f]
                    disabled:opacity-50 disabled:cursor-not-allowed
                    ${tool.enabled ? "bg-[#2f8fff]/60" : "bg-[#1a3670]"}
                  `}
                  role="switch"
                  aria-checked={tool.enabled}
                  aria-label={`${tool.enabled ? "Disable" : "Enable"} ${tool.name}`}
                >
                  <span
                    className={`
                      inline-block h-5 w-5 transform rounded-full bg-white shadow
                      transition-transform duration-200
                      ${tool.enabled ? "translate-x-6" : "translate-x-1"}
                    `}
                  />
                  {toggling[tool.name] && (
                    <span className="absolute inset-0 flex items-center justify-center">
                      <Loader2 className="h-4 w-4 animate-spin text-[#9bc3ff]" />
                    </span>
                  )}
                </button>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}

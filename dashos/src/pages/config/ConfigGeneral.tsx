import { useEffect, useState, type ReactNode } from "react";
import { Save, AlertTriangle, CheckCircle, FileCode2, List } from "lucide-react";
import { getConfig, putConfig } from "@/lib/api";
import { useWalletAuth } from "@/contexts/WalletAuthContext";

type ViewMode = "raw" | "list";

function renderConfigValue(value: unknown, depth = 0): ReactNode {
  if (value === null) return <span className="text-slate-500">null</span>;
  if (typeof value === "boolean") return <span className="text-amber-300">{String(value)}</span>;
  if (typeof value === "number") return <span className="text-emerald-300">{value}</span>;
  if (typeof value === "string") {
    return <span className="text-sky-200 break-all">{value}</span>;
  }
  if (Array.isArray(value)) {
    return (
      <ul className="mt-1 space-y-1 list-none pl-4 border-l border-slate-600/50">
        {value.map((item, i) => (
          <li key={i} className="flex gap-2">
            <span className="text-slate-500 shrink-0">[{i}]</span>
            {typeof item === "object" && item !== null && !Array.isArray(item) ? (
              <div className="flex-1 min-w-0">{renderConfigNode(item as Record<string, unknown>, depth + 1)}</div>
            ) : (
              renderConfigValue(item, depth + 1)
            )}
          </li>
        ))}
      </ul>
    );
  }
  if (typeof value === "object") {
    return renderConfigNode(value as Record<string, unknown>, depth + 1);
  }
  return <span className="text-slate-400">{String(value)}</span>;
}

function renderConfigNode(obj: Record<string, unknown>, depth = 0): ReactNode {
  const entries = Object.entries(obj);
  if (entries.length === 0) return <span className="text-slate-500">{"{}"}</span>;
  return (
    <div className={depth > 0 ? "pl-3 border-l border-slate-600/30" : ""}>
      {entries.map(([key, val]) => (
        <div key={key} className="py-1.5">
          <span className="text-electric-300 font-medium">{key}</span>
          <span className="text-slate-500 mx-1">:</span>
          {typeof val === "object" && val !== null && !Array.isArray(val) ? (
            <div className="mt-1">{renderConfigNode(val as Record<string, unknown>, depth + 1)}</div>
          ) : Array.isArray(val) ? (
            renderConfigValue(val, depth)
          ) : (
            renderConfigValue(val, depth)
          )}
        </div>
      ))}
    </div>
  );
}

function ConfigListView({ rawConfig }: { rawConfig: string }) {
  let parsed: Record<string, unknown>;
  try {
    parsed = JSON.parse(rawConfig || "{}") as Record<string, unknown>;
  } catch {
    return (
      <div className="p-4 text-rose-300 text-sm">
        Invalid JSON. Switch to raw view to fix.
      </div>
    );
  }
  return (
    <div className="p-4 font-mono text-sm space-y-0 min-h-[400px] overflow-auto">
      {renderConfigNode(parsed)}
    </div>
  );
}

export default function ConfigGeneral() {
  const { hasWriteAccess } = useWalletAuth();
  const [rawConfig, setRawConfig] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<ViewMode>("list");

  useEffect(() => {
    getConfig()
      .then((content) => setRawConfig(content))
      .catch((e) => setError(e instanceof Error ? e.message : "Load failed"))
      .finally(() => setLoading(false));
  }, []);

  const handleSave = async () => {
    if (!hasWriteAccess) {
      setError("Write access required. Connect wallet and sign.");
      return;
    }
    setSaving(true);
    setError(null);
    setSuccess(null);
    try {
      await putConfig(rawConfig);
      setSuccess("Config saved.");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving(false);
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
      <div className="sticky top-0 z-10 -mx-4 -mt-6 px-4 pt-6 pb-4 md:-mx-8 md:px-8 bg-[#020813]/95 backdrop-blur-sm flex items-center justify-between gap-4 flex-wrap shrink-0">
        <div className="flex items-center gap-3">
          <h2 className="text-base font-semibold text-white">Configuration</h2>
          <div className="flex rounded-lg border border-slate-600/60 bg-slate-900/50 p-0.5">
            <button
              type="button"
              onClick={() => setViewMode("list")}
              className={`flex items-center gap-2 px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
                viewMode === "list"
                  ? "bg-electric-500/20 text-electric-300"
                  : "text-slate-400 hover:text-slate-200"
              }`}
            >
              <List className="h-4 w-4" />
              List
            </button>
            <button
              type="button"
              onClick={() => setViewMode("raw")}
              className={`flex items-center gap-2 px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
                viewMode === "raw"
                  ? "bg-electric-500/20 text-electric-300"
                  : "text-slate-400 hover:text-slate-200"
              }`}
            >
              <FileCode2 className="h-4 w-4" />
              Raw
            </button>
          </div>
        </div>
        {viewMode === "raw" && (
          <button
            onClick={handleSave}
            disabled={saving || !hasWriteAccess}
            className="electric-button flex shrink-0 items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium disabled:opacity-50"
          >
            <Save className="h-4 w-4" />
            {saving ? "Saving…" : "Save"}
          </button>
        )}
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
                Connect your wallet and sign to edit configuration.
              </p>
            </div>
          </div>
        </div>
      )}

      {success && (
        <div className="electric-card p-3 border-emerald-500/30 bg-emerald-950/20 flex items-center gap-2">
          <CheckCircle className="h-4 w-4 text-emerald-400 flex-shrink-0" />
          <span className="text-sm text-emerald-300">{success}</span>
        </div>
      )}

      {error && (
        <div className="electric-card p-3 border-rose-500/30 bg-rose-950/20 flex items-center gap-2">
          <AlertTriangle className="h-4 w-4 text-rose-400 flex-shrink-0" />
          <span className="text-sm text-rose-300">{error}</span>
        </div>
      )}

      <div className="electric-card overflow-hidden">
        {viewMode === "raw" ? (
          <textarea
            value={rawConfig}
            onChange={(e) => setRawConfig(e.target.value)}
            disabled={!hasWriteAccess}
            placeholder="JSON configuration (automaton.json)..."
            className="w-full min-h-[400px] p-4 bg-[#05112c]/80 border-0 text-[#e2e8f0] font-mono text-sm resize-y focus:outline-none focus:ring-2 focus:ring-[#4f83ff] focus:ring-inset disabled:opacity-60"
            spellCheck={false}
          />
        ) : (
          <ConfigListView rawConfig={rawConfig} />
        )}
      </div>
    </div>
  );
}

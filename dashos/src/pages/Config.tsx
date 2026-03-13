import { useEffect, useState } from "react";
import { Settings, Save, AlertTriangle, CheckCircle } from "lucide-react";
import { getConfig, putConfig } from "@/lib/api";
import { useWalletAuth } from "@/contexts/WalletAuthContext";

export default function Config() {
  const { hasWriteAccess } = useWalletAuth();
  const [rawConfig, setRawConfig] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

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
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Settings className="h-5 w-5 text-[#9bc3ff]" />
          <h2 className="text-base font-semibold text-white">Configuration</h2>
        </div>
        <button
          onClick={handleSave}
          disabled={saving || !hasWriteAccess}
          className="electric-button flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium disabled:opacity-50"
        >
          <Save className="h-4 w-4" />
          {saving ? "Saving…" : "Save"}
        </button>
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
        <textarea
          value={rawConfig}
          onChange={(e) => setRawConfig(e.target.value)}
          disabled={!hasWriteAccess}
          placeholder="TOML configuration..."
          className="w-full min-h-[400px] p-4 bg-[#05112c]/80 border-0 text-[#e2e8f0] font-mono text-sm resize-y focus:outline-none focus:ring-2 focus:ring-[#4f83ff] focus:ring-inset disabled:opacity-60"
          spellCheck={false}
        />
      </div>
    </div>
  );
}

import { useEffect, useState } from "react";
import {
  Sparkles,
  AlertTriangle,
  Loader2,
  Save,
  Plus,
  Trash2,
  CheckCircle,
  Wand2,
} from "lucide-react";
import { useWalletAuth } from "@/contexts/WalletAuthContext";
import {
  getSoulConfig,
  putSoulConfig,
  postSoulEnhance,
  type SoulConfig,
} from "@/lib/api";

export default function ConfigSoul() {
  const { hasWriteAccess } = useWalletAuth();
  const [config, setConfig] = useState<SoulConfig>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [enhanceWords, setEnhanceWords] = useState("");
  const [enhancing, setEnhancing] = useState(false);

  useEffect(() => {
    getSoulConfig()
      .then(setConfig)
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
      await putSoulConfig(config);
      setSuccess("Soul config saved.");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving(false);
    }
  };

  const updateConstraint = (idx: number, value: string) => {
    const arr = [...(config.behavioralConstraints || [])];
    arr[idx] = value;
    setConfig((c) => ({ ...c, behavioralConstraints: arr }));
  };

  const addConstraint = () => {
    setConfig((c) => ({
      ...c,
      behavioralConstraints: [...(c.behavioralConstraints || []), ""],
    }));
  };

  const removeConstraint = (idx: number) => {
    const arr = [...(config.behavioralConstraints || [])];
    arr.splice(idx, 1);
    setConfig((c) => ({ ...c, behavioralConstraints: arr }));
  };

  const handleEnhance = async (apply: boolean) => {
    const words = enhanceWords.trim();
    if (!words) {
      setError("Enter a few words to enhance.");
      return;
    }
    if (words.split(/\s+/).filter(Boolean).length < 5) {
      setError("Enter at least 5 words to enhance.");
      return;
    }
    if (!hasWriteAccess && apply) {
      setError("Write access required to apply. Connect wallet and sign.");
      return;
    }
    setEnhancing(true);
    setError(null);
    setSuccess(null);
    try {
      const res = await postSoulEnhance(words, apply);
      setConfig((c) => ({ ...c, systemPrompt: res.systemPrompt }));
      if (apply) {
        setSuccess("System prompt enhanced and saved.");
        setEnhanceWords("");
      } else {
        setSuccess("Preview ready. Click 'Enhance & Apply' to save.");
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "Enhance failed");
    } finally {
      setEnhancing(false);
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
        <div className="flex items-center gap-3">
          <div className="electric-icon h-10 w-10 rounded-xl flex items-center justify-center">
            <Sparkles className="h-5 w-5 text-[#9bc3ff]" />
          </div>
          <div>
            <h2 className="text-lg font-semibold text-white">Soul</h2>
            <p className="text-sm text-[#8aa8df]">
              Agent personality, system prompt, tone, and behavioral constraints
            </p>
          </div>
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
                Connect your wallet and sign to edit soul configuration.
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

      <div className="electric-card p-6 space-y-6">
        <div className="rounded-lg border border-[#29509c] bg-[#071228]/50 p-4 space-y-3">
          <label className="block text-sm font-medium text-[#8aa8df]">
            Soul enhancer
          </label>
          <p className="text-xs text-[#6b8fcc]">
            Enter at least 5 casual words. The AI will turn them into a complete,
            ready-to-use system prompt.
          </p>
          <div className="flex gap-2">
            <input
              type="text"
              value={enhanceWords}
              onChange={(e) => setEnhanceWords(e.target.value)}
              disabled={!hasWriteAccess}
              placeholder="e.g. helpful financial assistant with warm tone and analytical style"
              className="flex-1 rounded-lg border border-[#29509c] bg-[#071228]/90 px-3 py-2 text-sm text-white placeholder:text-[#6b8fcc] focus:border-[#4f83ff] focus:outline-none disabled:opacity-60"
            />
            <button
              type="button"
              onClick={() => handleEnhance(false)}
              disabled={enhancing || !hasWriteAccess}
              className="electric-button flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium disabled:opacity-50"
            >
              {enhancing ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Wand2 className="h-4 w-4" />
              )}
              Preview
            </button>
            <button
              type="button"
              onClick={() => handleEnhance(true)}
              disabled={enhancing || !hasWriteAccess}
              className="electric-button flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium disabled:opacity-50"
            >
              {enhancing ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Wand2 className="h-4 w-4" />
              )}
              Enhance & Apply
            </button>
          </div>
        </div>

        <div>
          <label className="mb-2 block text-sm font-medium text-[#8aa8df]">
            System prompt
          </label>
          <p className="mb-1 text-xs text-[#6b8fcc]">
            Core instructions that define the agent&apos;s role and behavior.
          </p>
          <textarea
            value={config.systemPrompt ?? ""}
            onChange={(e) =>
              setConfig((c) => ({ ...c, systemPrompt: e.target.value }))
            }
            disabled={!hasWriteAccess}
            rows={4}
            placeholder="You are a helpful financial assistant..."
            className="w-full rounded-lg border border-[#29509c] bg-[#071228]/90 px-3 py-2 text-sm text-white placeholder:text-[#6b8fcc] focus:border-[#4f83ff] focus:outline-none disabled:opacity-60 resize-y"
          />
        </div>

        <div>
          <label className="mb-2 block text-sm font-medium text-[#8aa8df]">
            Personality
          </label>
          <p className="mb-1 text-xs text-[#6b8fcc]">
            Traits and characteristics (e.g. helpful, analytical, curious).
          </p>
          <input
            type="text"
            value={config.personality ?? ""}
            onChange={(e) =>
              setConfig((c) => ({ ...c, personality: e.target.value }))
            }
            disabled={!hasWriteAccess}
            placeholder="helpful, analytical, curious"
            className="w-full rounded-lg border border-[#29509c] bg-[#071228]/90 px-3 py-2 text-sm text-white placeholder:text-[#6b8fcc] focus:border-[#4f83ff] focus:outline-none disabled:opacity-60"
          />
        </div>

        <div>
          <label className="mb-2 block text-sm font-medium text-[#8aa8df]">
            Tone
          </label>
          <p className="mb-1 text-xs text-[#6b8fcc]">
            Communication style (e.g. professional, friendly, concise).
          </p>
          <input
            type="text"
            value={config.tone ?? ""}
            onChange={(e) =>
              setConfig((c) => ({ ...c, tone: e.target.value }))
            }
            disabled={!hasWriteAccess}
            placeholder="professional"
            className="w-full rounded-lg border border-[#29509c] bg-[#071228]/90 px-3 py-2 text-sm text-white placeholder:text-[#6b8fcc] focus:border-[#4f83ff] focus:outline-none disabled:opacity-60"
          />
        </div>

        <div>
          <label className="mb-2 block text-sm font-medium text-[#8aa8df]">
            Behavioral constraints
          </label>
          <p className="mb-2 text-xs text-[#6b8fcc]">
            Rules the agent must follow (e.g. never disclose private keys).
          </p>
          <div className="space-y-2">
            {(config.behavioralConstraints || []).map((c, idx) => (
              <div key={idx} className="flex gap-2">
                <input
                  type="text"
                  value={c}
                  onChange={(e) => updateConstraint(idx, e.target.value)}
                  disabled={!hasWriteAccess}
                  placeholder="e.g. Never disclose private keys"
                  className="flex-1 rounded-lg border border-[#29509c] bg-[#071228]/90 px-3 py-2 text-sm text-white placeholder:text-[#6b8fcc] focus:border-[#4f83ff] focus:outline-none disabled:opacity-60"
                />
                {hasWriteAccess && (
                  <button
                    type="button"
                    onClick={() => removeConstraint(idx)}
                    className="p-2 rounded text-[#6b8fcc] hover:text-rose-400 hover:bg-rose-950/30"
                    aria-label="Remove constraint"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                )}
              </div>
            ))}
            {hasWriteAccess && (
              <button
                type="button"
                onClick={addConstraint}
                className="flex items-center gap-2 px-3 py-2 rounded-lg border border-dashed border-[#29509c] text-[#6b8fcc] hover:text-[#9bc3ff] hover:border-[#4f83ff] transition-colors text-sm"
              >
                <Plus className="h-4 w-4" />
                Add constraint
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

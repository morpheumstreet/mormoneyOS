import { useEffect, useState } from "react";
import {
  Users,
  AlertTriangle,
  Loader2,
  ChevronDown,
  ChevronUp,
  Save,
} from "lucide-react";
import { useWalletAuth } from "@/contexts/WalletAuthContext";
import {
  getSocial,
  patchSocialEnabled,
  putSocialConfig,
  type SocialChannelItem,
  type SocialConfigField,
} from "@/lib/api";

const MASKED_PLACEHOLDER = "••••••••";

function formatArrayValue(val: unknown): string {
  if (Array.isArray(val)) return val.join(", ");
  if (typeof val === "string") return val;
  return "";
}

export default function ConfigSocial() {
  const { hasWriteAccess } = useWalletAuth();
  const [channels, setChannels] = useState<SocialChannelItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [toggling, setToggling] = useState<Record<string, boolean>>({});
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});
  const [formValues, setFormValues] = useState<Record<string, Record<string, string | boolean>>>({});
  const [saving, setSaving] = useState<Record<string, boolean>>({});

  useEffect(() => {
    getSocial()
      .then((res) => {
        setChannels(res.channels || []);
        const initial: Record<string, Record<string, string | boolean>> = {};
        for (const c of res.channels || []) {
          if (c.configFields && c.config) {
            const vals: Record<string, string | boolean> = {};
            for (const f of c.configFields) {
              const v = c.config[f.key];
              if (f.type === "boolean") {
                vals[f.key] = !!v;
              } else if (f.type === "array") {
                vals[f.key] = formatArrayValue(v);
              } else if (f.type === "password") {
                vals[f.key] = ""; // Never prefill password; placeholder shows "configured"
              } else {
                vals[f.key] = (typeof v === "string" ? v : "") || "";
              }
            }
            initial[c.name] = vals;
          }
        }
        setFormValues(initial);
      })
      .catch((e) => setError(e instanceof Error ? e.message : "Load failed"))
      .finally(() => setLoading(false));
  }, []);

  const handleToggle = async (channel: SocialChannelItem) => {
    if (!hasWriteAccess) return;
    const nextEnabled = !channel.enabled;
    setToggling((prev) => ({ ...prev, [channel.name]: true }));
    try {
      await patchSocialEnabled(channel.name, nextEnabled);
      setChannels((prev) =>
        prev.map((c) =>
          c.name === channel.name ? { ...c, enabled: nextEnabled } : c
        )
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : "Update failed");
    } finally {
      setToggling((prev) => ({ ...prev, [channel.name]: false }));
    }
  };

  const handleSaveConfig = async (channel: SocialChannelItem) => {
    if (!hasWriteAccess || !channel.configFields) return;
    setSaving((prev) => ({ ...prev, [channel.name]: true }));
    setError(null);
    try {
      const vals = formValues[channel.name] || {};
      const config: Record<string, unknown> = {};
      for (const f of channel.configFields) {
        const v = vals[f.key];
        if (f.type === "boolean") {
          config[f.key] = !!v;
        } else if (f.type === "array") {
          const str = typeof v === "string" ? v : "";
          config[f.key] = str
            ? str.split(",").map((s) => s.trim()).filter(Boolean)
            : [];
        } else if (f.type === "password") {
          if (typeof v === "string" && v !== "" && v !== MASKED_PLACEHOLDER) {
            config[f.key] = v;
          }
        } else {
          if (typeof v === "string" && v !== "") {
            config[f.key] = v;
          }
        }
      }
      const res = await putSocialConfig(channel.name, config);
      if (res.ok && res.validated) {
        const fresh = await getSocial();
        setChannels(fresh.channels || []);
        setExpanded((prev) => ({ ...prev, [channel.name]: false }));
      } else {
        setError(res.error || "Validation failed");
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving((prev) => ({ ...prev, [channel.name]: false }));
    }
  };

  const updateFormValue = (channelName: string, key: string, value: string | boolean) => {
    setFormValues((prev) => ({
      ...prev,
      [channelName]: {
        ...(prev[channelName] || {}),
        [key]: value,
      },
    }));
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
          <Users className="h-5 w-5 text-[#9bc3ff]" />
        </div>
        <div>
          <h2 className="text-lg font-semibold text-white">Social</h2>
          <p className="text-sm text-[#8aa8df]">
            Enable channels and configure API keys. Restart required to apply changes.
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
                Connect your wallet and sign to configure channels.
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
          {channels.length === 0 ? (
            <div className="px-4 py-8 text-center text-sm text-[#8aa8df]">
              No social channels available.
            </div>
          ) : (
            channels.map((channel) => (
              <div key={channel.name} className="px-4 py-3">
                <div className="flex items-center justify-between gap-4">
                  <button
                    type="button"
                    onClick={() =>
                      setExpanded((prev) => ({
                        ...prev,
                        [channel.name]: !prev[channel.name],
                      }))
                    }
                    className="flex min-w-0 flex-1 items-center gap-2 text-left"
                  >
                    {expanded[channel.name] ? (
                      <ChevronUp className="h-4 w-4 shrink-0 text-[#7ea5eb]" />
                    ) : (
                      <ChevronDown className="h-4 w-4 shrink-0 text-[#7ea5eb]" />
                    )}
                    <p className="font-medium text-white">
                      {channel.displayName || channel.name}
                    </p>
                    {!channel.ready && channel.enabled && (
                      <span className="text-xs text-amber-400/90">
                        Not configured
                      </span>
                    )}
                  </button>
                  <button
                    type="button"
                    onClick={() => handleToggle(channel)}
                    disabled={!hasWriteAccess || !!toggling[channel.name]}
                    className={`
                      relative inline-flex h-7 w-12 shrink-0 items-center rounded-full
                      transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-[#4f83ff] focus:ring-offset-2 focus:ring-offset-[#050d1f]
                      disabled:opacity-50 disabled:cursor-not-allowed
                      ${channel.enabled ? "bg-[#2f8fff]/60" : "bg-[#1a3670]"}
                    `}
                    role="switch"
                    aria-checked={channel.enabled}
                  >
                    <span
                      className={`
                        inline-block h-5 w-5 transform rounded-full bg-white shadow
                        transition-transform duration-200
                        ${channel.enabled ? "translate-x-6" : "translate-x-1"}
                      `}
                    />
                    {toggling[channel.name] && (
                      <span className="absolute inset-0 flex items-center justify-center">
                        <Loader2 className="h-4 w-4 animate-spin text-[#9bc3ff]" />
                      </span>
                    )}
                  </button>
                </div>

                {expanded[channel.name] && channel.configFields && channel.configFields.length > 0 && (
                  <div className="mt-4 space-y-3 border-t border-[#1a3670] pt-4">
                    {channel.configFields.map((field: SocialConfigField) => (
                      <div key={field.key}>
                        <label className="mb-1 block text-xs font-medium text-[#8aa8df]">
                          {field.label}
                          {field.required && (
                            <span className="ml-1 text-amber-400">*</span>
                          )}
                        </label>
                        {field.description && (
                          <p className="mb-1 text-xs text-[#6b8fcc]">
                            {field.description}
                          </p>
                        )}
                        {field.type === "boolean" ? (
                          <label className="flex items-center gap-2">
                            <input
                              type="checkbox"
                              checked={
                                (formValues[channel.name]?.[field.key] as boolean) ?? false
                              }
                              onChange={(e) =>
                                updateFormValue(
                                  channel.name,
                                  field.key,
                                  e.target.checked
                                )
                              }
                              disabled={!hasWriteAccess}
                              className="rounded border-[#29509c] bg-[#071228]/90 text-[#4f83ff] focus:ring-[#4f83ff]"
                            />
                            <span className="text-sm text-[#94a3b8]">
                              Enable
                            </span>
                          </label>
                        ) : field.type === "array" ? (
                          <input
                            type="text"
                            value={
                              (formValues[channel.name]?.[field.key] as string) ?? formatArrayValue(channel.config?.[field.key])
                            }
                            onChange={(e) =>
                              updateFormValue(channel.name, field.key, e.target.value)
                            }
                            disabled={!hasWriteAccess}
                            placeholder="Comma-separated values"
                            className="w-full rounded-lg border border-[#29509c] bg-[#071228]/90 px-3 py-2 text-sm text-white placeholder:text-[#6b8fcc] focus:border-[#4f83ff] focus:outline-none disabled:opacity-60"
                          />
                        ) : (
                          <input
                            type={field.type === "password" ? "password" : "text"}
                            value={
                              (formValues[channel.name]?.[field.key] as string) ?? ""
                            }
                            onChange={(e) =>
                              updateFormValue(channel.name, field.key, e.target.value)
                            }
                            disabled={!hasWriteAccess}
                            placeholder={
                              field.type === "password" && channel.config?.[field.key]
                                ? "•••••••• (leave blank to keep)"
                                : undefined
                            }
                            className="w-full rounded-lg border border-[#29509c] bg-[#071228]/90 px-3 py-2 text-sm text-white placeholder:text-[#6b8fcc] focus:border-[#4f83ff] focus:outline-none disabled:opacity-60"
                          />
                        )}
                      </div>
                    ))}
                    <button
                      type="button"
                      onClick={() => handleSaveConfig(channel)}
                      disabled={!hasWriteAccess || !!saving[channel.name]}
                      className="electric-button flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium disabled:opacity-50"
                    >
                      {saving[channel.name] ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <Save className="h-4 w-4" />
                      )}
                      {saving[channel.name] ? "Saving…" : "Save config"}
                    </button>
                  </div>
                )}
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}

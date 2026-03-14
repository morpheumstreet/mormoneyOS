import { useEffect, useState } from "react";
import {
  Network,
  AlertTriangle,
  Loader2,
  ChevronDown,
  ChevronUp,
  Save,
  RefreshCw,
  ExternalLink,
} from "lucide-react";
import { useWalletAuth } from "@/contexts/WalletAuthContext";
import {
  getTunnelProviders,
  getTunnels,
  putTunnelProvider,
  postTunnelProviderRestart,
  type TunnelProviderField,
  type TunnelProviderSchema,
} from "@/lib/api";

const MASKED_PLACEHOLDER = "••••••••";

const PROVIDER_LABELS: Record<string, string> = {
  bore: "Bore",
  localtunnel: "Localtunnel",
  cloudflare: "Cloudflare",
  ngrok: "ngrok",
  tailscale: "Tailscale",
  custom: "Custom",
};

export default function ConfigTunnel() {
  const { hasWriteAccess } = useWalletAuth();
  const [providersData, setProvidersData] = useState<{
    providers: string[];
    schemas: Record<string, TunnelProviderSchema>;
    config: { defaultProvider: string; providers: Record<string, Record<string, unknown>> };
  } | null>(null);
  const [tunnels, setTunnels] = useState<{ port: number; provider: string; public_url: string }[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});
  const [formValues, setFormValues] = useState<
    Record<string, Record<string, string | boolean>>
  >({});
  const [saving, setSaving] = useState<Record<string, boolean>>({});
  const [restarting, setRestarting] = useState<Record<string, boolean>>({});
  const [toggling, setToggling] = useState<Record<string, boolean>>({});

  const fetchData = async () => {
    setLoading(true);
    setError(null);
    try {
      const [provRes, tunnelsRes] = await Promise.all([
        getTunnelProviders(),
        getTunnels(),
      ]);
      setProvidersData(provRes);
      setTunnels(tunnelsRes.tunnels || []);

      const initial: Record<string, Record<string, string | boolean>> = {};
      const provConfig = provRes.config?.providers || {};
      const schemas = provRes.schemas || {};
      for (const name of provRes.providers || []) {
        const pc = provConfig[name] as Record<string, unknown> | undefined;
        const schema = schemas[name];
        if (schema?.fields) {
          const vals: Record<string, string | boolean> = {};
          for (const f of schema.fields) {
            const v = pc?.[f.name];
            if (f.type === "boolean") {
              vals[f.name] = !!v;
            } else if (f.type === "password") {
              vals[f.name] = ""; // Never prefill; placeholder when configured
            } else {
              vals[f.name] = (typeof v === "string" ? v : "") || "";
            }
          }
          initial[name] = vals;
        }
      }
      setFormValues(initial);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Load failed");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleSaveConfig = async (name: string) => {
    if (!hasWriteAccess || !providersData) return;
    const schema = providersData.schemas[name];
    if (!schema?.fields?.length) return;

    setSaving((prev) => ({ ...prev, [name]: true }));
    setError(null);
    try {
      const vals = formValues[name] || {};
      const body: Record<string, unknown> = {};
      const pc = providersData.config?.providers?.[name] as Record<string, unknown> | undefined;
      body.enabled = !!pc?.enabled;
      for (const f of schema.fields) {
        const v = vals[f.name];
        if (f.type === "boolean") {
          body[f.name] = !!v;
        } else if (f.type === "password") {
          if (typeof v === "string" && v !== "" && v !== MASKED_PLACEHOLDER) {
            body[f.name] = v;
          }
        } else {
          if (typeof v === "string" && v !== "") {
            body[f.name] = v;
          }
        }
      }
      await putTunnelProvider(name, body);
      await fetchData();
      setExpanded((prev) => ({ ...prev, [name]: false }));
    } catch (e) {
      setError(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving((prev) => ({ ...prev, [name]: false }));
    }
  };

  const handleToggleEnabled = async (name: string) => {
    if (!hasWriteAccess || !providersData) return;
    const pc = providersData.config?.providers?.[name] as Record<string, unknown> | undefined;
    const nextEnabled = !pc?.enabled;
    setToggling((prev) => ({ ...prev, [name]: true }));
    setError(null);
    try {
      await putTunnelProvider(name, { enabled: nextEnabled });
      await fetchData();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Update failed");
    } finally {
      setToggling((prev) => ({ ...prev, [name]: false }));
    }
  };

  const handleRestart = async (name: string) => {
    if (!hasWriteAccess) return;
    setRestarting((prev) => ({ ...prev, [name]: true }));
    setError(null);
    try {
      await postTunnelProviderRestart(name);
      await fetchData();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Restart failed");
    } finally {
      setRestarting((prev) => ({ ...prev, [name]: false }));
    }
  };

  const updateFormValue = (providerName: string, key: string, value: string | boolean) => {
    setFormValues((prev) => ({
      ...prev,
      [providerName]: {
        ...(prev[providerName] || {}),
        [key]: value,
      },
    }));
  };

  const isConfigured = (name: string, field: TunnelProviderField) => {
    if (field.type !== "password") return false;
    const pc = providersData?.config?.providers?.[name] as Record<string, unknown> | undefined;
    const v = pc?.[field.name];
    return typeof v === "string" && v !== ""; // Server returns "***" when configured
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
          <Network className="h-5 w-5 text-[#9bc3ff]" />
        </div>
        <div>
          <h2 className="text-lg font-semibold text-white">Tunnel</h2>
          <p className="text-sm text-[#8aa8df]">
            Configure tunnel providers (bore, cloudflare, ngrok, tailscale). API keys required for cloudflare, ngrok, tailscale.
          </p>
        </div>
      </div>

      {!hasWriteAccess && (
        <div className="electric-card p-4 border-amber-500/30 bg-amber-950/20">
          <div className="flex items-start gap-3">
            <AlertTriangle className="h-5 w-5 text-amber-400 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-amber-200">Write access required</p>
              <p className="text-sm text-amber-300/80 mt-1">
                Connect your wallet and sign to configure tunnel providers.
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

      {tunnels.length > 0 && (
        <div className="electric-card p-4">
          <h3 className="text-sm font-medium text-white mb-3">Active tunnels</h3>
          <div className="space-y-2">
            {tunnels.map((t) => (
              <div
                key={`${t.port}-${t.provider}`}
                className="flex items-center justify-between gap-4 rounded-lg border border-[#1a3670] bg-[#071228]/50 px-3 py-2"
              >
                <div className="min-w-0">
                  <p className="text-sm font-medium text-white">
                    Port {t.port} ({t.provider})
                  </p>
                  <a
                    href={t.public_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-xs text-[#7ea5eb] hover:underline flex items-center gap-1 truncate"
                  >
                    {t.public_url}
                    <ExternalLink className="h-3 w-3 flex-shrink-0" />
                  </a>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="electric-card overflow-hidden">
        <div className="divide-y divide-[#1a3670]">
          {!providersData?.providers?.length ? (
            <div className="px-4 py-8 text-center text-sm text-[#8aa8df]">
              No tunnel providers available.
            </div>
          ) : (
            providersData.providers.map((name) => {
              const schema = providersData.schemas[name];
              const pc = providersData.config?.providers?.[name] as Record<string, unknown> | undefined;
              const enabled = !!pc?.enabled;
              const hasAuthFields = schema?.fields?.some((f) => f.type === "password") ?? false;
              const needsConfig = hasAuthFields && !schema?.fields?.every((f) => {
                if (f.type !== "password") return true;
                const v = pc?.[f.name];
                return typeof v === "string" && v !== "" && v !== "***";
              });

              return (
                <div key={name} className="px-4 py-3">
                  <div className="flex items-center justify-between gap-4">
                    <button
                      type="button"
                      onClick={() =>
                        setExpanded((prev) => ({ ...prev, [name]: !prev[name] }))
                      }
                      className="flex min-w-0 flex-1 items-center gap-2 text-left"
                    >
                      {expanded[name] ? (
                        <ChevronUp className="h-4 w-4 shrink-0 text-[#7ea5eb]" />
                      ) : (
                        <ChevronDown className="h-4 w-4 shrink-0 text-[#7ea5eb]" />
                      )}
                      <p className="font-medium text-white">
                        {PROVIDER_LABELS[name] ?? name}
                      </p>
                      {needsConfig && (
                        <span className="text-xs text-amber-400/90">API key required</span>
                      )}
                    </button>
                    <button
                      type="button"
                      onClick={() => handleToggleEnabled(name)}
                      disabled={!hasWriteAccess || !!toggling[name]}
                      className={`
                        relative inline-flex h-7 w-12 shrink-0 items-center rounded-full
                        transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-[#4f83ff] focus:ring-offset-2 focus:ring-offset-[#050d1f]
                        disabled:opacity-50 disabled:cursor-not-allowed
                        ${enabled ? "bg-[#2f8fff]/60" : "bg-[#1a3670]"}
                      `}
                      role="switch"
                      aria-checked={enabled}
                    >
                      <span
                        className={`
                          inline-block h-5 w-5 transform rounded-full bg-white shadow
                          transition-transform duration-200
                          ${enabled ? "translate-x-6" : "translate-x-1"}
                        `}
                      />
                      {toggling[name] && (
                        <span className="absolute inset-0 flex items-center justify-center">
                          <Loader2 className="h-4 w-4 animate-spin text-[#9bc3ff]" />
                        </span>
                      )}
                    </button>
                    {["cloudflare", "ngrok", "tailscale"].includes(name) && (
                      <button
                        type="button"
                        onClick={() => handleRestart(name)}
                        disabled={!hasWriteAccess || !!restarting[name] || needsConfig}
                        title="Reload provider from config (requires API key in automaton.json)"
                        className="electric-button flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium disabled:opacity-50"
                      >
                        {restarting[name] ? (
                          <Loader2 className="h-3.5 w-3.5 animate-spin" />
                        ) : (
                          <RefreshCw className="h-3.5 w-3.5" />
                        )}
                        Restart
                      </button>
                    )}
                  </div>

                  {expanded[name] && schema?.fields && schema.fields.length > 0 && (
                    <div className="mt-4 space-y-3 border-t border-[#1a3670] pt-4">
                      {schema.fields.map((field: TunnelProviderField) => (
                        <div key={field.name}>
                          <label className="mb-1 block text-xs font-medium text-[#8aa8df]">
                            {field.label ?? field.name}
                            {field.required && (
                              <span className="ml-1 text-amber-400">*</span>
                            )}
                          </label>
                          {field.help && (
                            <p className="mb-1 text-xs text-[#6b8fcc]">{field.help}</p>
                          )}
                          {field.type === "boolean" ? (
                            <label className="flex items-center gap-2">
                              <input
                                type="checkbox"
                                checked={
                                  (formValues[name]?.[field.name] as boolean) ?? false
                                }
                                onChange={(e) =>
                                  updateFormValue(name, field.name, e.target.checked)
                                }
                                disabled={!hasWriteAccess}
                                className="rounded border-[#29509c] bg-[#071228]/90 text-[#4f83ff] focus:ring-[#4f83ff]"
                              />
                              <span className="text-sm text-[#94a3b8]">Enable</span>
                            </label>
                          ) : (
                            <input
                              type={field.type === "password" ? "password" : "text"}
                              value={
                                (formValues[name]?.[field.name] as string) ?? ""
                              }
                              onChange={(e) =>
                                updateFormValue(name, field.name, e.target.value)
                              }
                              disabled={!hasWriteAccess}
                              placeholder={
                                field.type === "password" && isConfigured(name, field)
                                  ? "•••••••• (leave blank to keep)"
                                  : field.type === "password"
                                    ? "Enter API key or use ${ENV_VAR}"
                                    : undefined
                              }
                              className="w-full rounded-lg border border-[#29509c] bg-[#071228]/90 px-3 py-2 text-sm text-white placeholder:text-[#6b8fcc] focus:border-[#4f83ff] focus:outline-none disabled:opacity-60"
                            />
                          )}
                        </div>
                      ))}
                      <button
                        type="button"
                        onClick={() => handleSaveConfig(name)}
                        disabled={!hasWriteAccess || !!saving[name]}
                        className="electric-button flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium disabled:opacity-50"
                      >
                        {saving[name] ? (
                          <Loader2 className="h-4 w-4 animate-spin" />
                        ) : (
                          <Save className="h-4 w-4" />
                        )}
                        {saving[name] ? "Saving…" : "Save config"}
                      </button>
                    </div>
                  )}

                  {expanded[name] && (!schema?.fields || schema.fields.length === 0) && (
                    <div className="mt-4 border-t border-[#1a3670] pt-4 text-sm text-[#8aa8df]">
                      No configuration required. Provider is ready when enabled.
                    </div>
                  )}
                </div>
              );
            })
          )}
        </div>
      </div>
    </div>
  );
}

import { Key, ChevronDown, ChevronRight, Loader2 } from "lucide-react";
import { inputSm } from "@/lib/theme";
import type { ModelProvider } from "@/lib/api";

function getEndpointPlaceholder(providerKey: string): string {
  switch (providerKey) {
    case "ollama":
      return "http://localhost:11434";
    case "azure":
      return "https://YOUR_RESOURCE.openai.azure.com/openai/deployments/YOUR_DEPLOYMENT";
    case "vertex":
      return "https://REGION-aiplatform.googleapis.com/v1/projects/PROJECT/locations/REGION";
    default:
      return "https://...";
  }
}

interface ProviderApiKeysPanelProps {
  hasWriteAccess: boolean;
  providers: ModelProvider[];
  apiKeysOpen: boolean;
  setApiKeysOpen: (open: boolean | ((prev: boolean) => boolean)) => void;
  providerKeyValues: Record<string, string>;
  setProviderKeyValues: React.Dispatch<
    React.SetStateAction<Record<string, string>>
  >;
  savingProviderKey: string | null;
  saveProviderKey: (
    configKey: string,
    value: string,
    hasWriteAccess: boolean,
    setError: (s: string | null) => void,
    load: () => void
  ) => Promise<void>;
  saveProviderEndpoint: (
    providerKey: string,
    url: string,
    hasWriteAccess: boolean,
    setError: (s: string | null) => void,
    load: () => void
  ) => Promise<void>;
  setError: (s: string | null) => void;
  load: () => void;
}

export function ProviderApiKeysPanel({
  hasWriteAccess,
  providers,
  apiKeysOpen,
  setApiKeysOpen,
  providerKeyValues,
  setProviderKeyValues,
  savingProviderKey,
  saveProviderKey,
  saveProviderEndpoint,
  setError,
  load,
}: ProviderApiKeysPanelProps) {
  if (!hasWriteAccess) return null;

  const providerGroups = [
    {
      label: "Resellers (aggregate others' models)",
      list: providers.filter((p) => p.configKey && !p.local && p.isReseller),
    },
    {
      label: "Direct (source model developers)",
      list: providers.filter((p) => p.configKey && !p.local && !p.isReseller),
    },
  ];

  return (
    <div className="electric-card overflow-hidden">
      <button
        type="button"
        onClick={() => setApiKeysOpen((o) => !o)}
        className="w-full flex items-center justify-between px-4 py-3 text-left hover:bg-[#07132f]/50 transition-colors"
      >
        <div className="flex items-center gap-2">
          <Key className="h-4 w-4 text-[#9bc3ff]" />
          <span className="text-sm font-medium text-white">
            Provider API keys
          </span>
          <span className="text-xs text-[#6b8fcc]">
            {providers.filter((p) => p.configKey && !p.local).length} providers
          </span>
        </div>
        {apiKeysOpen ? (
          <ChevronDown className="h-4 w-4 text-[#6b8fcc]" />
        ) : (
          <ChevronRight className="h-4 w-4 text-[#6b8fcc]" />
        )}
      </button>
      {apiKeysOpen && (
        <div className="px-4 pb-4 pt-0 border-t border-[#1a3670]">
          <p className="text-xs text-[#8aa8df] mt-3 mb-3">
            Add API keys.{" "}
            <strong className="text-[#9bc3ff]">Resellers</strong> (OpenRouter,
            Together, Fireworks) aggregate models from multiple developers.{" "}
            <strong className="text-[#9bc3ff]">Direct</strong> = source model
            developers.
          </p>
          <div className="space-y-4">
            {providerGroups.map(
              ({ label, list }) =>
                list.length > 0 && (
                  <div key={label}>
                    <p className="text-[10px] font-medium text-[#6b8fcc] uppercase tracking-wider mb-2">
                      {label}
                    </p>
                    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                      {list.map((p) => (
                        <div key={p.key} className="space-y-1.5">
                          <label className="block text-xs font-medium text-[#8aa8df]">
                            {p.displayName}
                            {p.hasKey && (
                              <span className="ml-1.5 text-emerald-400 text-[10px]">
                                ✓
                              </span>
                            )}
                          </label>
                          <div className="flex gap-2">
                            <input
                              type="password"
                              value={providerKeyValues[p.configKey!] ?? ""}
                              onChange={(e) =>
                                setProviderKeyValues((prev) => ({
                                  ...prev,
                                  [p.configKey!]: e.target.value,
                                }))
                              }
                              placeholder={p.hasKey ? "••••••••" : "sk-..."}
                              className={`flex-1 ${inputSm}`}
                            />
                            <button
                              type="button"
                              onClick={() =>
                                saveProviderKey(
                                  p.configKey!,
                                  providerKeyValues[p.configKey!] ?? "",
                                  hasWriteAccess,
                                  setError,
                                  load
                                )
                              }
                              disabled={
                                savingProviderKey === p.configKey ||
                                (providerKeyValues[p.configKey!] ?? "").trim() ===
                                  ""
                              }
                              className="electric-button px-2 py-1.5 text-xs shrink-0"
                            >
                              {savingProviderKey === p.configKey ? (
                                <Loader2 className="h-3 w-3 animate-spin" />
                              ) : (
                                "Save"
                              )}
                            </button>
                          </div>
                          {p.endpointConfigKey && (
                            <div className="space-y-1">
                              <label className="block text-xs text-[#6b8fcc]">
                                Endpoint URL
                              </label>
                              <div className="flex gap-2">
                                <input
                                  type="text"
                                  value={
                                    providerKeyValues[p.endpointConfigKey] ?? ""
                                  }
                                  onChange={(e) =>
                                    setProviderKeyValues((prev) => ({
                                      ...prev,
                                      [p.endpointConfigKey!]: e.target.value,
                                    }))
                                  }
                                  placeholder={getEndpointPlaceholder(p.key)}
                                  className={`flex-1 ${inputSm}`}
                                />
                                <button
                                  type="button"
                                  onClick={() =>
                                    saveProviderEndpoint(
                                      p.key,
                                      providerKeyValues[p.endpointConfigKey!] ??
                                        "",
                                      hasWriteAccess,
                                      setError,
                                      load
                                    )
                                  }
                                  disabled={
                                    savingProviderKey === p.endpointConfigKey
                                  }
                                  className="electric-button px-2 py-1.5 text-xs shrink-0"
                                >
                                  {savingProviderKey === p.endpointConfigKey ? (
                                    <Loader2 className="h-3 w-3 animate-spin" />
                                  ) : (
                                    "Save"
                                  )}
                                </button>
                              </div>
                            </div>
                          )}
                        </div>
                      ))}
                    </div>
                  </div>
                )
            )}
          </div>
        </div>
      )}
    </div>
  );
}

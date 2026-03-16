import { useState, useEffect, useCallback, useRef } from "react";
import { Plus, Loader2, Check, AlertCircle } from "lucide-react";
import { FormInput } from "@/components/ui/FormInput";
import { FormSelect } from "@/components/ui/FormSelect";
import { FormCheckbox } from "@/components/ui/FormCheckbox";
import type { ModelProvider } from "@/lib/api";
import { getLocalProviderModels } from "@/lib/api";
import {
  eligibleProviders,
  getEndpointConfigKey,
  LOCAL_PROVIDERS,
  DEFAULT_LOCAL_URLS,
  DEFAULT_CONTEXT_LIMIT,
  DEFAULT_COST_CAP_CENTS,
} from "./constants";
import { inputSm } from "@/lib/theme";
import type { AddFormState } from "./useModels";

/** Check if modelId matches any of the model IDs (supports prefix match for Ollama tags like llama3.2:latest) */
function modelMatches(modelId: string, ids: string[]): boolean {
  const trimmed = modelId.trim().toLowerCase();
  if (!trimmed) return false;
  return ids.some((id) => {
    const idLower = id.toLowerCase();
    return idLower === trimmed || idLower.startsWith(trimmed + ":") || trimmed.startsWith(idLower);
  });
}

interface AddModelFormProps {
  hasWriteAccess: boolean;
  providers: ModelProvider[];
  form: AddFormState;
  setForm: React.Dispatch<React.SetStateAction<AddFormState>>;
  localProviderUrl: string;
  setLocalProviderUrl: React.Dispatch<React.SetStateAction<string>>;
  catalogType: "cloud" | "local";
  adding: boolean;
  onAdd: (options?: { localProviderUrl?: string }) => void | Promise<void>;
  providerKeyValues: Record<string, string>;
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

export function AddModelForm({
  hasWriteAccess,
  providers,
  form,
  setForm,
  localProviderUrl,
  setLocalProviderUrl,
  catalogType,
  adding,
  onAdd,
  providerKeyValues,
  saveProviderEndpoint,
  setError,
  load,
}: AddModelFormProps) {
  const showLocalForm = catalogType === "local";
  const isLocalProvider = LOCAL_PROVIDERS.includes(form.provider as (typeof LOCAL_PROVIDERS)[number]);
  const endpointKey = getEndpointConfigKey(form.provider, providers);
  const savedUrl = endpointKey ? providerKeyValues[endpointKey] : undefined;
  const defaultUrl = DEFAULT_LOCAL_URLS[form.provider] ?? "";
  const [fetchingModels, setFetchingModels] = useState(false);
  const [detectedModels, setDetectedModels] = useState<{
    active: string[];
    available: string[];
  } | null>(null);
  const fetchAbortRef = useRef<AbortController | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Sync URL from saved config when provider changes
  useEffect(() => {
    const url = savedUrl || defaultUrl;
    setLocalProviderUrl(url);
    setDetectedModels(null);
  }, [form.provider, savedUrl, defaultUrl]);

  // Fetch models when URL changes (debounced)
  const fetchModels = useCallback(
    (url: string) => {
      if (!url.trim() || !LOCAL_PROVIDERS.includes(form.provider as (typeof LOCAL_PROVIDERS)[number])) {
        setDetectedModels(null);
        return;
      }
      if (fetchAbortRef.current) fetchAbortRef.current.abort();
      fetchAbortRef.current = new AbortController();
      setFetchingModels(true);
      setDetectedModels(null);
      getLocalProviderModels(form.provider, url.trim())
        .then((res) => {
          setDetectedModels({
            active: res.activeModels ?? [],
            available: res.availableModels ?? [],
          });
        })
        .catch(() => {
          setDetectedModels(null);
        })
        .finally(() => {
          setFetchingModels(false);
          fetchAbortRef.current = null;
        });
    },
    [form.provider]
  );

  useEffect(() => {
    if (!isLocalProvider) return;
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      fetchModels(localProviderUrl);
      debounceRef.current = null;
    }, 400);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [localProviderUrl, isLocalProvider, fetchModels]);

  const allModels = detectedModels
    ? [...new Set([...detectedModels.active, ...detectedModels.available])]
    : [];
  const modelValid = form.modelId.trim()
    ? modelMatches(form.modelId, allModels)
    : null;
  const modelActive = form.modelId.trim() && detectedModels
    ? modelMatches(form.modelId, detectedModels.active)
    : false;

  const handleAdd = useCallback(async () => {
    if (isLocalProvider && localProviderUrl.trim()) {
      const currentSaved = endpointKey ? providerKeyValues[endpointKey] : undefined;
      const effectiveUrl = localProviderUrl.trim();
      if (endpointKey && effectiveUrl !== (currentSaved || DEFAULT_LOCAL_URLS[form.provider])) {
        setError(null);
        try {
          await saveProviderEndpoint(form.provider, effectiveUrl, hasWriteAccess, setError, load);
        } catch {
          return;
        }
      }
    }
    await onAdd({ localProviderUrl: isLocalProvider ? localProviderUrl.trim() : undefined });
  }, [
    isLocalProvider,
    localProviderUrl,
    endpointKey,
    providerKeyValues,
    form.provider,
    saveProviderEndpoint,
    hasWriteAccess,
    setError,
    load,
    onAdd,
  ]);

  if (!hasWriteAccess) return null;

  return (
    <div className="electric-card p-4">
      <div className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between xl:gap-6">
        <div className="xl:shrink-0">
          <h3 className="text-sm font-medium text-white">Add model</h3>
          <p className="text-xs text-[#8aa8df] mt-0.5">
            {showLocalForm
              ? "Connect a local provider. Enter URL or path, then pick a model."
              : "Pick from the Catalog tab, or enter manually."}
          </p>
        </div>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4 xl:flex xl:flex-1 xl:flex-wrap xl:items-end xl:gap-4 xl:justify-start">
          <FormSelect
            label="Provider"
            value={form.provider}
            onChange={(e) => {
              const next = e.target.value;
              const nextLocal = LOCAL_PROVIDERS.includes(next as (typeof LOCAL_PROVIDERS)[number]);
              setForm((f) => ({
                ...f,
                provider: next,
                modelId: "",
                contextLimit: nextLocal ? 0 : DEFAULT_CONTEXT_LIMIT,
                costCapCents: nextLocal ? 0 : DEFAULT_COST_CAP_CENTS,
              }));
            }}
            className="xl:w-[140px] xl:min-w-[140px]"
          >
            {eligibleProviders(providers)
              .filter((p) => p.local === showLocalForm)
              .map((p) => (
                <option key={p.key} value={p.key}>
                  {p.displayName || p.key}
                </option>
              ))}
          </FormSelect>

          {showLocalForm ? (
            <>
              <div className="sm:col-span-2 lg:col-span-1 xl:min-w-[200px] xl:flex-1 xl:max-w-[280px]">
                <label className="mb-1 block text-xs font-medium text-[#8aa8df]">
                  Endpoint URL or path
                </label>
                <div className="flex gap-2 items-center">
                  <input
                    type="url"
                    value={localProviderUrl}
                    onChange={(e) => setLocalProviderUrl(e.target.value)}
                    placeholder={defaultUrl || "http://localhost:..."}
                    className={`flex-1 ${inputSm}`}
                  />
                  {fetchingModels && (
                    <Loader2 className="h-4 w-4 animate-spin text-[#6b8fcc] shrink-0" />
                  )}
                </div>
                <p className="mt-0.5 text-[10px] text-[#6b8fcc]">
                  Default: {defaultUrl || "—"}
                </p>
              </div>

              <div className="sm:col-span-2 lg:col-span-1 xl:min-w-[200px] xl:flex-1 xl:max-w-[280px]">
                {allModels.length > 0 ? (
                  <FormSelect
                    label="Model"
                    value={form.modelId}
                    onChange={(e) =>
                      setForm((f) => ({ ...f, modelId: e.target.value }))
                    }
                    className="w-full"
                  >
                    <option value="">Select a model</option>
                    {allModels.map((id) => (
                      <option key={id} value={id}>
                        {id}
                        {detectedModels?.active.includes(id) ? " (loaded)" : ""}
                      </option>
                    ))}
                  </FormSelect>
                ) : (
                  <FormInput
                    label="Model"
                    value={form.modelId}
                    onChange={(e) =>
                      setForm((f) => ({ ...f, modelId: e.target.value }))
                    }
                    placeholder="Enter URL to fetch models, or type model ID"
                  />
                )}
                {form.modelId.trim() && modelValid !== null && (
                  <p
                    className="mt-0.5 flex items-center gap-1 text-xs"
                    title={
                      modelValid
                        ? modelActive
                          ? "Model is loaded and ready"
                          : "Model is available"
                        : "Model not found at endpoint"
                    }
                  >
                    {modelValid ? (
                      <>
                        <Check className="h-3.5 w-3.5 text-emerald-400 shrink-0" />
                        <span className={modelActive ? "text-emerald-400" : "text-[#8aa8df]"}>
                          {modelActive ? "Loaded" : "Available"}
                        </span>
                      </>
                    ) : (
                      <>
                        <AlertCircle className="h-3.5 w-3.5 text-amber-400 shrink-0" />
                        <span className="text-amber-400">Not found</span>
                      </>
                    )}
                  </p>
                )}
              </div>

              <FormCheckbox
                label="Enabled"
                checked={form.enabled}
                onChange={(e) =>
                  setForm((f) => ({ ...f, enabled: e.target.checked }))
                }
                className="pb-2 xl:pb-0"
              />
              <button
                type="button"
                onClick={handleAdd}
                disabled={
                  adding ||
                  !localProviderUrl.trim() ||
                  !form.modelId.trim() ||
                  fetchingModels ||
                  modelValid === false
                }
                className="electric-button flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium disabled:opacity-50 shrink-0 sm:col-span-2 lg:col-span-4"
              >
                {adding ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Plus className="h-4 w-4" />
                )}
                {adding ? "Adding…" : "Add"}
              </button>
            </>
          ) : (
            <>
              <div className="sm:col-span-2 lg:col-span-1 xl:min-w-[200px] xl:flex-1 xl:max-w-[280px]">
                <FormInput
                  label="Model ID"
                  value={form.modelId}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, modelId: e.target.value }))
                  }
                  placeholder="e.g. gpt-4o, llama-3.3-70b-versatile"
                />
              </div>

              <FormInput
                label="Context"
                type="number"
                value={form.contextLimit}
                onChange={(e) =>
                  setForm((f) => ({
                    ...f,
                    contextLimit: parseInt(e.target.value, 10) || 0,
                  }))
                }
                min={1}
                className="xl:w-[90px] xl:min-w-[90px]"
              />
              <FormInput
                label="Cost cap (¢)"
                type="number"
                value={form.costCapCents}
                onChange={(e) =>
                  setForm((f) => ({
                    ...f,
                    costCapCents: parseInt(e.target.value, 10) || 0,
                  }))
                }
                min={0}
                className="xl:w-[90px] xl:min-w-[90px]"
              />
              <FormCheckbox
                label="Enabled"
                checked={form.enabled}
                onChange={(e) =>
                  setForm((f) => ({ ...f, enabled: e.target.checked }))
                }
                className="pb-2 xl:pb-0"
              />
              <button
                type="button"
                onClick={handleAdd}
                disabled={adding || !form.modelId.trim()}
                className="electric-button flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium disabled:opacity-50 shrink-0 sm:col-span-2 lg:col-span-4"
              >
                {adding ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Plus className="h-4 w-4" />
                )}
                {adding ? "Adding…" : "Add"}
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

import { useState, useEffect, useCallback, useRef } from "react";
import { ChevronUp, ChevronDown, Pencil, Trash2, Check, X, Loader2, Cpu, Cloud, AlertCircle, Timer } from "lucide-react";
import { FormCheckbox } from "@/components/ui/FormCheckbox";
import { inputSm } from "@/lib/theme";
import type { ModelItem, ModelProvider } from "@/lib/api";
import { getLocalProviderModels, postModelsTestLatency } from "@/lib/api";
import type { EditFormState } from "./useModels";
import { LOCAL_PROVIDERS, getEndpointConfigKey, DEFAULT_LOCAL_URLS } from "./constants";

function modelMatches(modelId: string, ids: string[]): boolean {
  const trimmed = modelId.trim().toLowerCase();
  if (!trimmed) return false;
  return ids.some((id) => {
    const idLower = id.toLowerCase();
    return idLower === trimmed || idLower.startsWith(trimmed + ":") || trimmed.startsWith(idLower);
  });
}

function isLocalModel(model: ModelItem, providers: ModelProvider[]): boolean {
  const p = providers.find((x) => x.key === model.provider);
  return !!(p?.local ?? LOCAL_PROVIDERS.includes(model.provider as (typeof LOCAL_PROVIDERS)[number]));
}

interface ModelListItemProps {
  model: ModelItem;
  index: number;
  totalModels: number;
  providers: ModelProvider[];
  hasWriteAccess: boolean;
  editing: boolean;
  deleting: boolean;
  editForm: EditFormState | undefined;
  onEditFormChange: (updates: Partial<EditFormState>) => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
  onStartEdit: () => void;
  onSaveEdit: () => void;
  onCancelEdit: () => void;
  onDelete: () => void;
  /** For local models: endpoint URL editing (same logic as AddModelForm) */
  providerKeyValues?: Record<string, string>;
  saveProviderEndpoint?: (
    providerKey: string,
    url: string,
    hasWriteAccess: boolean,
    setError: (s: string | null) => void,
    load: () => void
  ) => Promise<void>;
  setError?: (s: string | null) => void;
  load?: () => void;
  savingProviderKey?: string | null;
}

export function ModelListItem({
  model,
  index,
  totalModels,
  providers,
  hasWriteAccess,
  editing,
  deleting,
  editForm,
  onEditFormChange,
  onMoveUp,
  onMoveDown,
  onStartEdit,
  onSaveEdit,
  onCancelEdit,
  onDelete,
  providerKeyValues = {},
  saveProviderEndpoint,
  setError,
  load,
  savingProviderKey = null,
}: ModelListItemProps) {
  const providerName =
    providers.find((p) => p.key === model.provider)?.displayName ?? model.provider;
  const isLocal = isLocalModel(model, providers);

  const endpointKey = getEndpointConfigKey(model.provider, providers);
  const savedUrl = endpointKey ? providerKeyValues[endpointKey] : undefined;
  const defaultUrl = DEFAULT_LOCAL_URLS[model.provider] ?? "";

  const [editEndpointUrl, setEditEndpointUrl] = useState("");
  const [fetchingModels, setFetchingModels] = useState(false);
  const [testingLatency, setTestingLatency] = useState(false);
  const [latencyMs, setLatencyMs] = useState<number | null>(null);
  const latencyClearRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [detectedModels, setDetectedModels] = useState<{
    active: string[];
    available: string[];
  } | null>(null);
  const fetchAbortRef = useRef<AbortController | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (editing && isLocal) {
      setEditEndpointUrl(savedUrl || defaultUrl);
      setDetectedModels(null);
    }
  }, [editing, isLocal, savedUrl, defaultUrl]);

  const fetchModels = useCallback(
    (url: string) => {
      if (!url.trim() || !LOCAL_PROVIDERS.includes(model.provider as (typeof LOCAL_PROVIDERS)[number])) {
        setDetectedModels(null);
        return;
      }
      if (fetchAbortRef.current) fetchAbortRef.current.abort();
      fetchAbortRef.current = new AbortController();
      setFetchingModels(true);
      setDetectedModels(null);
      getLocalProviderModels(model.provider, url.trim())
        .then((res) => {
          setDetectedModels({
            active: res.activeModels ?? [],
            available: res.availableModels ?? [],
          });
        })
        .catch(() => setDetectedModels(null))
        .finally(() => {
          setFetchingModels(false);
          fetchAbortRef.current = null;
        });
    },
    [model.provider]
  );

  useEffect(() => {
    if (!editing || !isLocal) return;
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      fetchModels(editEndpointUrl);
      debounceRef.current = null;
    }, 400);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [editing, isLocal, editEndpointUrl, fetchModels]);

  useEffect(() => () => {
    if (latencyClearRef.current) {
      clearTimeout(latencyClearRef.current);
    }
  }, []);

  const allModels = detectedModels
    ? [...new Set([...detectedModels.active, ...detectedModels.available])]
    : [];
  const currentModelId = editForm?.modelId ?? model.modelId;
  const modelValid = currentModelId.trim()
    ? modelMatches(currentModelId, allModels)
    : null;
  const modelActive = Boolean(
    currentModelId.trim() &&
      detectedModels &&
      modelMatches(currentModelId, detectedModels.active)
  );

  const handleSaveLocal = useCallback(async () => {
    if (!saveProviderEndpoint || !setError || !load) {
      onSaveEdit();
      return;
    }
    const effectiveUrl = editEndpointUrl.trim();
    const currentSaved = endpointKey ? providerKeyValues[endpointKey] : undefined;
    const urlChanged =
      endpointKey &&
      effectiveUrl &&
      effectiveUrl !== (currentSaved || defaultUrl);

    if (urlChanged) {
      setError(null);
      try {
        await saveProviderEndpoint(model.provider, effectiveUrl, hasWriteAccess, setError, load);
      } catch {
        return;
      }
    }
    onSaveEdit();
  }, [
    editEndpointUrl,
    endpointKey,
    providerKeyValues,
    defaultUrl,
    model.provider,
    saveProviderEndpoint,
    setError,
    load,
    hasWriteAccess,
    onSaveEdit,
  ]);

  const isSavingEndpoint = Boolean(
    endpointKey && savingProviderKey === endpointKey
  );

  const effectiveUrl = (endpointKey ? providerKeyValues[endpointKey] : undefined) || defaultUrl;

  const handleTestLatency = useCallback(async () => {
    const url = effectiveUrl?.trim();
    const modelId = model.modelId?.trim();
    if (!url || !modelId) {
      setError?.(url ? "Model ID required" : "Endpoint URL required");
      return;
    }
    if (latencyClearRef.current) {
      clearTimeout(latencyClearRef.current);
      latencyClearRef.current = null;
    }
    setError?.(null);
    setLatencyMs(null);
    setTestingLatency(true);
    try {
      const res = await postModelsTestLatency(model.provider, url, modelId);
      setLatencyMs(res.latencyMs);
      latencyClearRef.current = setTimeout(() => {
        setLatencyMs(null);
        latencyClearRef.current = null;
      }, 60_000);
    } catch (e) {
      setError?.(e instanceof Error ? e.message : "Latency test failed");
    } finally {
      setTestingLatency(false);
    }
  }, [model.provider, model.modelId, effectiveUrl, setError]);

  return (
    <div className="flex items-center gap-3 px-4 py-3 hover:bg-[#07132f]/50 transition-colors group">
      {hasWriteAccess && (
        <div className="flex flex-col gap-0.5 shrink-0">
          <button
            type="button"
            onClick={onMoveUp}
            disabled={index === 0}
            className="p-0.5 rounded text-[#6b8fcc] hover:text-[#9bc3ff] disabled:opacity-30 disabled:cursor-not-allowed"
            aria-label="Move up"
          >
            <ChevronUp className="h-4 w-4" />
          </button>
          <button
            type="button"
            onClick={onMoveDown}
            disabled={index === totalModels - 1}
            className="p-0.5 rounded text-[#6b8fcc] hover:text-[#9bc3ff] disabled:opacity-30 disabled:cursor-not-allowed"
            aria-label="Move down"
          >
            <ChevronDown className="h-4 w-4" />
          </button>
        </div>
      )}

      <div className="min-w-0 flex-1">
        {editing ? (
          isLocal ? (
            /* Local model edit layout — endpoint URL + model ID + context + enabled (same logic as AddModelForm) */
            <div className="flex flex-col gap-2">
              <div className="flex flex-wrap items-center gap-3">
                <div className="flex items-center gap-2 shrink-0">
                  <Cpu className="h-4 w-4 text-emerald-400/80" aria-hidden />
                  <input
                    type="text"
                    value={editForm?.modelId ?? model.modelId}
                    onChange={(e) => onEditFormChange({ modelId: e.target.value })}
                    placeholder="Model ID"
                    className={`min-w-[140px] ${inputSm}`}
                  />
                </div>
                <div className="flex items-center gap-2 min-w-[200px] flex-1">
                  <label className="text-xs text-[#6b8fcc] shrink-0">
                    Endpoint URL
                  </label>
                  <input
                    type="url"
                    value={editEndpointUrl}
                    onChange={(e) => setEditEndpointUrl(e.target.value)}
                    placeholder={defaultUrl || "http://localhost:..."}
                    className={`flex-1 min-w-0 ${inputSm}`}
                  />
                  {fetchingModels && (
                    <Loader2 className="h-4 w-4 animate-spin text-[#6b8fcc] shrink-0" />
                  )}
                </div>
                <input
                  type="number"
                  value={editForm?.contextLimit ?? model.contextLimit ?? ""}
                  onChange={(e) =>
                    onEditFormChange({
                      contextLimit: parseInt(e.target.value, 10) || undefined,
                    })
                  }
                  placeholder="Context"
                  className={`w-20 ${inputSm}`}
                />
                {(editForm?.modelId ?? model.modelId).trim() && modelValid !== null && (
                  <span
                    className={`flex items-center gap-1 text-xs shrink-0 ${
                      modelValid ? (modelActive ? "text-emerald-400" : "text-[#8aa8df]") : "text-amber-400"
                    }`}
                    title={
                      modelValid
                        ? modelActive
                          ? "Model is loaded and ready"
                          : "Model is available"
                        : "Model not found at endpoint"
                    }
                  >
                    {modelValid ? (
                      <Check className="h-3.5 w-3.5 shrink-0" />
                    ) : (
                      <AlertCircle className="h-3.5 w-3.5 shrink-0" />
                    )}
                    {modelValid ? (modelActive ? "Loaded" : "Available") : "Not found"}
                  </span>
                )}
                <FormCheckbox
                  label="Enabled"
                  checked={!!(editForm?.enabled ?? model.enabled ?? true)}
                  onChange={(e) =>
                    onEditFormChange({ enabled: e.target.checked })
                  }
                />
                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    onClick={handleSaveLocal}
                    disabled={isSavingEndpoint}
                    className="electric-button flex items-center gap-1 px-2 py-1 rounded text-xs"
                  >
                    {isSavingEndpoint ? (
                      <Loader2 className="h-3 w-3 animate-spin" />
                    ) : (
                      <Check className="h-3 w-3" />
                    )}
                    Save
                  </button>
                  <button
                    type="button"
                    onClick={onCancelEdit}
                    className="flex items-center gap-1 px-2 py-1 rounded text-xs border border-[#29509c] text-[#8aa8df] hover:bg-[#07132f]/50"
                  >
                    <X className="h-3 w-3" />
                    Cancel
                  </button>
                </div>
              </div>
              <p className="text-[10px] text-[#6b8fcc]">
                Default: {defaultUrl || "—"}
              </p>
            </div>
          ) : (
            /* Cloud model edit layout */
            <div className="flex flex-wrap items-center gap-3">
              <div className="flex items-center gap-2 shrink-0">
                <Cloud className="h-4 w-4 text-blue-400/80" aria-hidden />
                <input
                  type="text"
                  value={editForm?.modelId ?? model.modelId}
                  onChange={(e) => onEditFormChange({ modelId: e.target.value })}
                  placeholder="Model ID"
                  className={`min-w-[140px] ${inputSm}`}
                />
              </div>
              <input
                type="password"
                value={editForm?.apiKey ?? ""}
                onChange={(e) =>
                  onEditFormChange({ apiKey: e.target.value })
                }
                placeholder="API key (leave blank to keep)"
                className={`min-w-[140px] ${inputSm}`}
              />
              <input
                type="number"
                value={editForm?.contextLimit ?? model.contextLimit ?? ""}
                onChange={(e) =>
                  onEditFormChange({
                    contextLimit: parseInt(e.target.value, 10) || undefined,
                  })
                }
                placeholder="Context"
                className={`w-20 ${inputSm}`}
              />
              <input
                type="number"
                value={editForm?.costCapCents ?? model.costCapCents ?? ""}
                onChange={(e) =>
                  onEditFormChange({
                    costCapCents: parseInt(e.target.value, 10) || undefined,
                  })
                }
                placeholder="¢"
                className={`w-16 ${inputSm}`}
              />
              <FormCheckbox
                label="Enabled"
                checked={!!(editForm?.enabled ?? model.enabled ?? true)}
                onChange={(e) =>
                  onEditFormChange({ enabled: e.target.checked })
                }
              />
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={onSaveEdit}
                  className="electric-button flex items-center gap-1 px-2 py-1 rounded text-xs"
                >
                  <Check className="h-3 w-3" />
                  Save
                </button>
                <button
                  type="button"
                  onClick={onCancelEdit}
                  className="flex items-center gap-1 px-2 py-1 rounded text-xs border border-[#29509c] text-[#8aa8df] hover:bg-[#07132f]/50"
                >
                  <X className="h-3 w-3" />
                  Cancel
                </button>
              </div>
            </div>
          )
        ) : isLocal ? (
          /* Local model view layout */
          <div className="flex flex-wrap items-center gap-3">
            <div className="flex items-center gap-2 shrink-0">
              <Cpu className="h-4 w-4 text-emerald-400/80" aria-hidden />
              <span className="font-medium text-white">{model.modelId}</span>
            </div>
            <span className="text-xs text-[#6b8fcc]">({providerName})</span>
            {endpointKey && (savedUrl || defaultUrl) && (
              <span className="text-xs text-[#8aa8df]" title={savedUrl || defaultUrl}>
                {savedUrl ? (
                  <>URL: {(savedUrl.length > 36 ? savedUrl.slice(0, 33) + "…" : savedUrl)}</>
                ) : (
                  <>Default: {defaultUrl}</>
                )}
              </span>
            )}
            {model.contextLimit != null && model.contextLimit > 0 && (
              <span className="text-xs text-[#8aa8df]">
                Context: {model.contextLimit.toLocaleString()}
              </span>
            )}
            {!model.enabled && (
              <span className="text-xs text-amber-400/90">Disabled</span>
            )}
            <span className="text-xs text-[#6b8fcc]">
              Priority: {model.priority ?? index}
            </span>
            {latencyMs != null && (
              <span className="text-xs text-emerald-400/90">
                {latencyMs} ms
              </span>
            )}
          </div>
        ) : (
          /* Cloud model view layout */
          <>
            <div className="flex items-center gap-2 flex-wrap">
              <Cloud className="h-4 w-4 text-blue-400/80 shrink-0" aria-hidden />
              <span className="font-medium text-white">{model.modelId}</span>
              <span className="text-xs text-[#6b8fcc]">({providerName})</span>
              {!model.enabled && (
                <span className="text-xs text-amber-400/90">Disabled</span>
              )}
            </div>
            <div className="mt-0.5 flex items-center gap-3 text-xs text-[#8aa8df]">
              {model.apiKeyMasked && (
                <span>API key: {model.apiKeyMasked}</span>
              )}
              {model.contextLimit != null && (
                <span>
                  Context: {model.contextLimit.toLocaleString()}
                </span>
              )}
              {model.costCapCents != null && (
                <span>Cap: {model.costCapCents}¢</span>
              )}
              <span>Priority: {model.priority ?? index}</span>
              {latencyMs != null && (
                <span className="text-emerald-400/90">{latencyMs} ms</span>
              )}
            </div>
          </>
        )}
      </div>

      {!editing && (
        <div className="flex items-center gap-1 shrink-0">
          {isLocal && effectiveUrl && (
            <button
              type="button"
              onClick={handleTestLatency}
              disabled={testingLatency}
              className="p-2 rounded text-[#6b8fcc] hover:text-[#9bc3ff] hover:bg-[#07132f]/50 disabled:opacity-50"
              aria-label="Test latency"
              title={latencyMs != null ? `${latencyMs} ms` : "Test latency"}
            >
              {testingLatency ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Timer className="h-4 w-4" />
              )}
            </button>
          )}
          {hasWriteAccess && (
            <>
              <button
                type="button"
                onClick={onStartEdit}
                className="p-2 rounded text-[#6b8fcc] hover:text-[#9bc3ff] hover:bg-[#07132f]/50"
                aria-label="Edit"
              >
                <Pencil className="h-4 w-4" />
              </button>
              <button
                type="button"
                onClick={onDelete}
                disabled={!!deleting}
                className="p-2 rounded text-[#6b8fcc] hover:text-rose-400 hover:bg-rose-950/30 disabled:opacity-50"
                aria-label="Delete"
              >
                {deleting ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Trash2 className="h-4 w-4" />
                )}
              </button>
            </>
          )}
        </div>
      )}
    </div>
  );
}

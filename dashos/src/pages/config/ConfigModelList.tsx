import { useEffect, useState } from "react";
import {
  Cpu,
  AlertTriangle,
  Loader2,
  Plus,
  Trash2,
  ChevronUp,
  ChevronDown,
  Pencil,
  X,
  Check,
} from "lucide-react";
import { useWalletAuth } from "@/contexts/WalletAuthContext";
import {
  getModels,
  postModel,
  patchModel,
  deleteModel,
  putModelsOrder,
  type ModelItem,
  type ModelProvider,
} from "@/lib/api";

const DEFAULT_CONTEXT_LIMIT = 8192;
const DEFAULT_COST_CAP_CENTS = 500;

export default function ConfigModelList() {
  const { hasWriteAccess } = useWalletAuth();
  const [models, setModels] = useState<ModelItem[]>([]);
  const [providers, setProviders] = useState<ModelProvider[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [adding, setAdding] = useState(false);
  const [editing, setEditing] = useState<string | null>(null);
  const [deleting, setDeleting] = useState<string | null>(null);
  const [form, setForm] = useState({
    provider: "",
    modelId: "",
    apiKey: "",
    contextLimit: DEFAULT_CONTEXT_LIMIT,
    costCapCents: DEFAULT_COST_CAP_CENTS,
    enabled: true,
  });
  const [editForm, setEditForm] = useState<
    Record<
      string,
      Partial<{
        modelId: string;
        apiKey: string;
        contextLimit: number;
        costCapCents: number;
        enabled: boolean;
      }>
    >
  >({});

  const load = () => {
    setLoading(true);
    getModels()
      .then((res) => {
        setModels(res.models || []);
        setProviders(res.providers || []);
        if (res.providers?.length && !form.provider) {
          setForm((f) => ({ ...f, provider: res.providers![0].key }));
        }
      })
      .catch((e) => setError(e instanceof Error ? e.message : "Load failed"))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
  }, []);

  const handleAdd = async () => {
    if (!hasWriteAccess || !form.provider || !form.modelId) return;
    setAdding(true);
    setError(null);
    try {
      await postModel({
        provider: form.provider,
        modelId: form.modelId.trim(),
        apiKey: form.apiKey.trim() || undefined,
        contextLimit: form.contextLimit,
        costCapCents: form.costCapCents,
        enabled: form.enabled,
      });
      setForm({
        provider: form.provider,
        modelId: "",
        apiKey: "",
        contextLimit: DEFAULT_CONTEXT_LIMIT,
        costCapCents: DEFAULT_COST_CAP_CENTS,
        enabled: true,
      });
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Add failed");
    } finally {
      setAdding(false);
    }
  };

  const handleUpdate = async (model: ModelItem) => {
    if (!hasWriteAccess) return;
    const raw = editForm[model.id];
    if (!raw) {
      setEditing(null);
      return;
    }
    const payload = { ...raw };
    if (payload.apiKey === "") delete payload.apiKey;
    if (Object.keys(payload).length === 0) {
      setEditing(null);
      return;
    }
    setEditing(null);
    setError(null);
    try {
      await patchModel(model.id, payload);
      setEditForm((prev) => {
        const next = { ...prev };
        delete next[model.id];
        return next;
      });
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Update failed");
    }
  };

  const handleDelete = async (model: ModelItem) => {
    if (!hasWriteAccess) return;
    setDeleting(model.id);
    setError(null);
    try {
      await deleteModel(model.id);
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Delete failed");
    } finally {
      setDeleting(null);
    }
  };

  const handleMove = async (model: ModelItem, direction: "up" | "down") => {
    if (!hasWriteAccess) return;
    const idx = models.findIndex((m) => m.id === model.id);
    if (idx < 0) return;
    const newIdx = direction === "up" ? idx - 1 : idx + 1;
    if (newIdx < 0 || newIdx >= models.length) return;
    const reordered = [...models];
    [reordered[idx], reordered[newIdx]] = [reordered[newIdx], reordered[idx]];
    const ids = reordered.map((m) => m.id);
    setError(null);
    try {
      await putModelsOrder(ids);
      setModels(reordered);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Reorder failed");
    }
  };

  const startEdit = (model: ModelItem) => {
    setEditing(model.id);
    setEditForm((prev) => ({
      ...prev,
      [model.id]: {
        modelId: model.modelId,
        apiKey: "",
        contextLimit: model.contextLimit ?? DEFAULT_CONTEXT_LIMIT,
        costCapCents: model.costCapCents ?? DEFAULT_COST_CAP_CENTS,
        enabled: model.enabled ?? true,
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
          <Cpu className="h-5 w-5 text-[#9bc3ff]" />
        </div>
        <div>
          <h2 className="text-lg font-semibold text-white">Model List</h2>
          <p className="text-sm text-[#8aa8df]">
            Add, remove, and prioritize LLM providers. Set API keys, model IDs,
            context limits, and cost caps.
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
                Connect your wallet and sign to manage models.
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

      {/* Add model form */}
      {hasWriteAccess && (
        <div className="electric-card p-4 space-y-4">
          <h3 className="text-sm font-medium text-white">Add model</h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-[#8aa8df]">
                Provider
              </label>
              <select
                value={form.provider}
                onChange={(e) =>
                  setForm((f) => ({ ...f, provider: e.target.value }))
                }
                className="w-full rounded-lg border border-[#29509c] bg-[#071228]/90 px-3 py-2 text-sm text-white focus:border-[#4f83ff] focus:outline-none"
              >
                {providers.map((p) => (
                  <option key={p.key} value={p.key}>
                    {p.displayName || p.key}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-[#8aa8df]">
                Model ID
              </label>
              <input
                type="text"
                value={form.modelId}
                onChange={(e) =>
                  setForm((f) => ({ ...f, modelId: e.target.value }))
                }
                placeholder="e.g. llama-3.3-70b-versatile"
                className="w-full rounded-lg border border-[#29509c] bg-[#071228]/90 px-3 py-2 text-sm text-white placeholder:text-[#6b8fcc] focus:border-[#4f83ff] focus:outline-none"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-[#8aa8df]">
                API key
              </label>
              <input
                type="password"
                value={form.apiKey}
                onChange={(e) =>
                  setForm((f) => ({ ...f, apiKey: e.target.value }))
                }
                placeholder="sk-..."
                className="w-full rounded-lg border border-[#29509c] bg-[#071228]/90 px-3 py-2 text-sm text-white placeholder:text-[#6b8fcc] focus:border-[#4f83ff] focus:outline-none"
              />
            </div>
            <div className="flex items-end gap-2">
              <div className="flex-1 grid grid-cols-2 gap-2">
                <div>
                  <label className="mb-1 block text-xs font-medium text-[#8aa8df]">
                    Context
                  </label>
                  <input
                    type="number"
                    value={form.contextLimit}
                    onChange={(e) =>
                      setForm((f) => ({
                        ...f,
                        contextLimit: parseInt(e.target.value, 10) || 0,
                      }))
                    }
                    min={1}
                    className="w-full rounded-lg border border-[#29509c] bg-[#071228]/90 px-3 py-2 text-sm text-white focus:border-[#4f83ff] focus:outline-none"
                  />
                </div>
                <div>
                  <label className="mb-1 block text-xs font-medium text-[#8aa8df]">
                    Cost cap (¢)
                  </label>
                  <input
                    type="number"
                    value={form.costCapCents}
                    onChange={(e) =>
                      setForm((f) => ({
                        ...f,
                        costCapCents: parseInt(e.target.value, 10) || 0,
                      }))
                    }
                    min={0}
                    className="w-full rounded-lg border border-[#29509c] bg-[#071228]/90 px-3 py-2 text-sm text-white focus:border-[#4f83ff] focus:outline-none"
                  />
                </div>
              </div>
              <label className="flex items-center gap-2 pb-2">
                <input
                  type="checkbox"
                  checked={form.enabled}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, enabled: e.target.checked }))
                  }
                  className="rounded border-[#29509c] bg-[#071228]/90 text-[#4f83ff] focus:ring-[#4f83ff]"
                />
                <span className="text-xs text-[#8aa8df]">Enabled</span>
              </label>
              <button
                type="button"
                onClick={handleAdd}
                disabled={adding || !form.modelId.trim()}
                className="electric-button flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium disabled:opacity-50 shrink-0"
              >
                {adding ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Plus className="h-4 w-4" />
                )}
                {adding ? "Adding…" : "Add"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Model list */}
      <div className="electric-card overflow-hidden">
        <div className="divide-y divide-[#1a3670]">
          {models.length === 0 ? (
            <div className="px-4 py-8 text-center text-sm text-[#8aa8df]">
              No models configured. Add one above to get started.
            </div>
          ) : (
            models.map((model, idx) => (
              <div
                key={model.id}
                className="flex items-center gap-3 px-4 py-3 hover:bg-[#07132f]/50 transition-colors group"
              >
                {hasWriteAccess && (
                  <div className="flex flex-col gap-0.5 shrink-0">
                    <button
                      type="button"
                      onClick={() => handleMove(model, "up")}
                      disabled={idx === 0}
                      className="p-0.5 rounded text-[#6b8fcc] hover:text-[#9bc3ff] disabled:opacity-30 disabled:cursor-not-allowed"
                      aria-label="Move up"
                    >
                      <ChevronUp className="h-4 w-4" />
                    </button>
                    <button
                      type="button"
                      onClick={() => handleMove(model, "down")}
                      disabled={idx === models.length - 1}
                      className="p-0.5 rounded text-[#6b8fcc] hover:text-[#9bc3ff] disabled:opacity-30 disabled:cursor-not-allowed"
                      aria-label="Move down"
                    >
                      <ChevronDown className="h-4 w-4" />
                    </button>
                  </div>
                )}

                <div className="min-w-0 flex-1">
                  {editing === model.id ? (
                    <div className="space-y-2">
                      <div className="flex flex-wrap gap-2">
                        <input
                          type="text"
                          value={editForm[model.id]?.modelId ?? model.modelId}
                          onChange={(e) =>
                            setEditForm((prev) => ({
                              ...prev,
                              [model.id]: {
                                ...prev[model.id],
                                modelId: e.target.value,
                              },
                            }))
                          }
                          placeholder="Model ID"
                          className="flex-1 min-w-[120px] rounded border border-[#29509c] bg-[#071228]/90 px-2 py-1.5 text-sm text-white focus:border-[#4f83ff] focus:outline-none"
                        />
                        <input
                          type="password"
                          value={editForm[model.id]?.apiKey ?? ""}
                          onChange={(e) =>
                            setEditForm((prev) => ({
                              ...prev,
                              [model.id]: {
                                ...prev[model.id],
                                apiKey: e.target.value,
                              },
                            }))
                          }
                          placeholder="API key (leave blank to keep)"
                          className="flex-1 min-w-[120px] rounded border border-[#29509c] bg-[#071228]/90 px-2 py-1.5 text-sm text-white placeholder:text-[#6b8fcc] focus:border-[#4f83ff] focus:outline-none"
                        />
                        <input
                          type="number"
                          value={
                            editForm[model.id]?.contextLimit ??
                            model.contextLimit ??
                            ""
                          }
                          onChange={(e) =>
                            setEditForm((prev) => ({
                              ...prev,
                              [model.id]: {
                                ...prev[model.id],
                                contextLimit: parseInt(e.target.value, 10) || undefined,
                              },
                            }))
                          }
                          placeholder="Context"
                          className="w-20 rounded border border-[#29509c] bg-[#071228]/90 px-2 py-1.5 text-sm text-white focus:border-[#4f83ff] focus:outline-none"
                        />
                        <input
                          type="number"
                          value={
                            editForm[model.id]?.costCapCents ??
                            model.costCapCents ??
                            ""
                          }
                          onChange={(e) =>
                            setEditForm((prev) => ({
                              ...prev,
                              [model.id]: {
                                ...prev[model.id],
                                costCapCents: parseInt(e.target.value, 10) || undefined,
                              },
                            }))
                          }
                          placeholder="¢"
                          className="w-16 rounded border border-[#29509c] bg-[#071228]/90 px-2 py-1.5 text-sm text-white focus:border-[#4f83ff] focus:outline-none"
                        />
                        <label className="flex items-center gap-1.5">
                          <input
                            type="checkbox"
                            checked={
                              editForm[model.id]?.enabled ?? model.enabled ?? true
                            }
                            onChange={(e) =>
                              setEditForm((prev) => ({
                                ...prev,
                                [model.id]: {
                                  ...prev[model.id],
                                  enabled: e.target.checked,
                                },
                              }))
                            }
                            className="rounded border-[#29509c] bg-[#071228]/90 text-[#4f83ff]"
                          />
                          <span className="text-xs text-[#8aa8df]">On</span>
                        </label>
                        <button
                          type="button"
                          onClick={() => handleUpdate(model)}
                          className="electric-button flex items-center gap-1 px-2 py-1 rounded text-xs"
                        >
                          <Check className="h-3 w-3" />
                          Save
                        </button>
                        <button
                          type="button"
                          onClick={() => {
                            setEditing(null);
                            setEditForm((prev) => {
                              const next = { ...prev };
                              delete next[model.id];
                              return next;
                            });
                          }}
                          className="flex items-center gap-1 px-2 py-1 rounded text-xs border border-[#29509c] text-[#8aa8df] hover:bg-[#07132f]/50"
                        >
                          <X className="h-3 w-3" />
                          Cancel
                        </button>
                      </div>
                    </div>
                  ) : (
                    <>
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="font-medium text-white">
                          {model.modelId}
                        </span>
                        <span className="text-xs text-[#6b8fcc]">
                          ({providers.find((p) => p.key === model.provider)?.displayName ?? model.provider})
                        </span>
                        {!model.enabled && (
                          <span className="text-xs text-amber-400/90">
                            Disabled
                          </span>
                        )}
                      </div>
                      <div className="mt-0.5 flex items-center gap-3 text-xs text-[#8aa8df]">
                        {model.apiKeyMasked && (
                          <span>API key: {model.apiKeyMasked}</span>
                        )}
                        {model.contextLimit != null && (
                          <span>Context: {model.contextLimit.toLocaleString()}</span>
                        )}
                        {model.costCapCents != null && (
                          <span>Cap: {model.costCapCents}¢</span>
                        )}
                        <span>Priority: {model.priority ?? idx}</span>
                      </div>
                    </>
                  )}
                </div>

                {editing !== model.id && hasWriteAccess && (
                  <div className="flex items-center gap-1 shrink-0">
                    <button
                      type="button"
                      onClick={() => startEdit(model)}
                      className="p-2 rounded text-[#6b8fcc] hover:text-[#9bc3ff] hover:bg-[#07132f]/50"
                      aria-label="Edit"
                    >
                      <Pencil className="h-4 w-4" />
                    </button>
                    <button
                      type="button"
                      onClick={() => handleDelete(model)}
                      disabled={!!deleting}
                      className="p-2 rounded text-[#6b8fcc] hover:text-rose-400 hover:bg-rose-950/30 disabled:opacity-50"
                      aria-label="Delete"
                    >
                      {deleting === model.id ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <Trash2 className="h-4 w-4" />
                      )}
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

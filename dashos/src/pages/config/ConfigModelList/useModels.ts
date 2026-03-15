import { useState, useCallback, type MutableRefObject } from "react";
import {
  getModels,
  postModel,
  patchModel,
  deleteModel,
  putModelsOrder,
  type ModelItem,
  type ModelProvider,
  type ModelCatalogEntry,
  type ModelsResponse,
} from "@/lib/api";
import { handleApiError } from "@/lib/api-error";
import {
  DEFAULT_CONTEXT_LIMIT,
  DEFAULT_COST_CAP_CENTS,
  eligibleProviders,
} from "./constants";

export interface AddFormState {
  provider: string;
  modelId: string;
  contextLimit: number;
  costCapCents: number;
  enabled: boolean;
}

export interface EditFormState {
  modelId?: string;
  apiKey?: string;
  contextLimit?: number;
  costCapCents?: number;
  enabled?: boolean;
}

export function useModels(
  initialProvider?: string,
  onLoadSuccessRef?: MutableRefObject<((res: ModelsResponse) => void) | null>
) {
  const [models, setModels] = useState<ModelItem[]>([]);
  const [providers, setProviders] = useState<ModelProvider[]>([]);
  const [catalog, setCatalog] = useState<ModelCatalogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [adding, setAdding] = useState(false);
  const [editing, setEditing] = useState<string | null>(null);
  const [deleting, setDeleting] = useState<string | null>(null);
  const [form, setForm] = useState<AddFormState>({
    provider: initialProvider ?? "",
    modelId: "",
    contextLimit: DEFAULT_CONTEXT_LIMIT,
    costCapCents: DEFAULT_COST_CAP_CENTS,
    enabled: true,
  });
  const [editForm, setEditForm] = useState<Record<string, EditFormState>>({});

  const load = useCallback(() => {
    setLoading(true);
    getModels()
      .then((res) => {
        setModels(res.models || []);
        setProviders(res.providers || []);
        setCatalog(res.catalog || []);
        setForm((f) => {
          const eligible = eligibleProviders(res.providers ?? []);
          if (!eligible.length) return f;
          const currentValid = eligible.some((p) => p.key === f.provider);
          if (!f.provider || !currentValid) {
            return { ...f, provider: eligible[0].key };
          }
          return f;
        });
        onLoadSuccessRef?.current?.(res);
      })
      .catch((e) => handleApiError(e, setError, "Load failed"))
      .finally(() => setLoading(false));
  }, [onLoadSuccessRef]);

  const handleAdd = useCallback(
    async (hasWriteAccess: boolean) => {
      if (!hasWriteAccess || !form.provider || !form.modelId) return;
      setAdding(true);
      setError(null);
      try {
        await postModel({
          provider: form.provider,
          modelId: form.modelId.trim(),
          contextLimit: form.contextLimit,
          costCapCents: form.costCapCents,
          enabled: form.enabled,
        });
        setForm({
          provider: form.provider,
          modelId: "",
          contextLimit: DEFAULT_CONTEXT_LIMIT,
          costCapCents: DEFAULT_COST_CAP_CENTS,
          enabled: true,
        });
        load();
      } catch (e) {
        handleApiError(e, setError, "Add failed");
      } finally {
        setAdding(false);
      }
    },
    [form, load]
  );

  const handleUpdate = useCallback(
    async (model: ModelItem, hasWriteAccess: boolean) => {
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
        handleApiError(e, setError, "Update failed");
      }
    },
    [editForm, load]
  );

  const handleDelete = useCallback(
    async (model: ModelItem, hasWriteAccess: boolean) => {
      if (!hasWriteAccess) return;
      setDeleting(model.id);
      setError(null);
      try {
        await deleteModel(model.id);
        load();
      } catch (e) {
        handleApiError(e, setError, "Delete failed");
      } finally {
        setDeleting(null);
      }
    },
    [load]
  );

  const handleMove = useCallback(
    async (model: ModelItem, direction: "up" | "down", hasWriteAccess: boolean) => {
      if (!hasWriteAccess) return;
      const idx = models.findIndex((m) => m.id === model.id);
      if (idx < 0) return;
      const newIdx = direction === "up" ? idx - 1 : idx + 1;
      if (newIdx < 0 || newIdx >= models.length) return;
      const reordered = [...models];
      [reordered[idx], reordered[newIdx]] = [reordered[newIdx], reordered[idx]];
      setError(null);
      try {
        await putModelsOrder(reordered.map((m) => m.id));
        setModels(reordered);
      } catch (e) {
        handleApiError(e, setError, "Reorder failed");
      }
    },
    [models]
  );

  const startEdit = useCallback((model: ModelItem) => {
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
  }, []);

  const cancelEdit = useCallback((modelId: string) => {
    setEditing(null);
    setEditForm((prev) => {
      const next = { ...prev };
      delete next[modelId];
      return next;
    });
  }, []);

  const pickFromCatalog = useCallback((entry: ModelCatalogEntry) => {
    setForm((f) => {
      const eligible = eligibleProviders(providers);
      const provider = eligible.some((p) => p.key === entry.provider)
        ? entry.provider
        : eligible[0]?.key ?? f.provider;
      return {
        ...f,
        provider,
        modelId: entry.modelId,
        contextLimit:
          entry.contextK > 0 ? entry.contextK * 1024 : DEFAULT_CONTEXT_LIMIT,
      };
    });
  }, [providers]);

  return {
    models,
    providers,
    catalog,
    loading,
    error,
    setError,
    adding,
    editing,
    deleting,
    form,
    setForm,
    editForm,
    setEditForm,
    load,
    handleAdd,
    handleUpdate,
    handleDelete,
    handleMove,
    startEdit,
    cancelEdit,
    pickFromCatalog,
  };
}

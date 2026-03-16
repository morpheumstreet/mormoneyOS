import { useCallback, useEffect, useState, useRef } from "react";
import { Cpu, AlertTriangle, Key, Search, Plus, Package } from "lucide-react";
import { useWalletAuth } from "@/contexts/WalletAuthContext";
import type { ModelsResponse } from "@/lib/api";
import { useModels } from "./useModels";
import { useCatalogFilters } from "./useCatalogFilters";
import { eligibleProviders, DEFAULT_CONTEXT_LIMIT, DEFAULT_COST_CAP_CENTS, DEFAULT_LOCAL_URLS } from "./constants";
import { useProviderKeys } from "./useProviderKeys";
import { ProviderApiKeysPanel } from "./ProviderApiKeysPanel";
import { ModelCatalog } from "./ModelCatalog";
import { AddModelForm } from "./AddModelForm";
import { ModelListItem } from "./ModelListItem";

export default function ConfigModelList() {
  const { hasWriteAccess } = useWalletAuth();
  const mergeEndpointRef = useRef<((res: ModelsResponse) => void) | null>(null);

  type SubTab = "provider-keys" | "catalog" | "add-model" | "model-shelf";
  const [subTab, setSubTab] = useState<SubTab>("catalog");
  const [catalogType, setCatalogType] = useState<"cloud" | "local">("cloud");
  const [catalogTierFilter, setCatalogTierFilter] = useState("all");
  const [catalogUseCaseFilter, setCatalogUseCaseFilter] = useState("all");
  const [catalogSort, setCatalogSort] = useState("params");
  const [catalogQuery, setCatalogQuery] = useState("");

  const modelsState = useModels(undefined, mergeEndpointRef);
  const providerKeys = useProviderKeys(modelsState.providers);
  mergeEndpointRef.current = providerKeys.mergeEndpointValuesFromResponse;

  const filteredCatalog = useCatalogFilters(
    modelsState.catalog,
    {
      type: catalogType,
      query: catalogQuery,
      tier: catalogTierFilter,
      useCase: catalogUseCaseFilter,
      sort: catalogSort,
    }
  );

  useEffect(() => {
    modelsState.load();
  }, [modelsState.load]);

  // Single handler: update catalog type and sync Add form in one place (avoids effect + extra render)
  const handleCatalogTypeChange = useCallback(
    (t: "cloud" | "local") => {
      setCatalogType(t);
      const eligible = eligibleProviders(modelsState.providers);
      const localProviders = eligible.filter((p) => p.local);
      const cloudProviders = eligible.filter((p) => !p.local);
      const currentProvider = modelsState.form.provider;
      const currentIsLocal = localProviders.some((p) => p.key === currentProvider);

      if (t === "local" && localProviders.length > 0 && !currentIsLocal) {
        const first = localProviders[0];
        modelsState.setForm((f) => ({
          ...f,
          provider: first.key,
          modelId: "",
          contextLimit: 0,
          costCapCents: 0,
        }));
        modelsState.setLocalProviderUrl(DEFAULT_LOCAL_URLS[first.key] ?? "");
      } else if (t === "cloud" && cloudProviders.length > 0 && currentIsLocal) {
        const first = cloudProviders[0];
        modelsState.setForm((f) => ({
          ...f,
          provider: first.key,
          modelId: "",
          contextLimit: DEFAULT_CONTEXT_LIMIT,
          costCapCents: DEFAULT_COST_CAP_CENTS,
        }));
      }
    },
    [
      modelsState.providers,
      modelsState.form.provider,
      modelsState.setForm,
      modelsState.setLocalProviderUrl,
    ]
  );

  if (modelsState.loading) {
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

      {modelsState.error && (
        <div className="electric-card p-3 border-rose-500/30 bg-rose-950/20 flex items-center gap-2">
          <AlertTriangle className="h-4 w-4 text-rose-400 flex-shrink-0" />
          <span className="text-sm text-rose-300">{modelsState.error}</span>
        </div>
      )}

      <div className="flex items-center gap-1 border-b border-[#1a3670]">
        {(
          [
            { id: "provider-keys" as const, icon: Key, label: "Provider API keys" },
            { id: "catalog" as const, icon: Search, label: "Catalog" },
            { id: "add-model" as const, icon: Plus, label: "Add Model" },
            { id: "model-shelf" as const, icon: Package, label: "Model Shelf" },
          ] as const
        ).map(({ id, icon: Icon, label }) => (
          <button
            key={id}
            type="button"
            onClick={() => setSubTab(id)}
            className={`flex items-center gap-2 px-4 py-2.5 text-sm font-medium transition-all ${
              subTab === id
                ? "border-b-2 border-[#4f83ff] text-white"
                : "text-[#8aa8df] hover:text-white hover:bg-[#07132f]/50"
            }`}
          >
            <Icon className="h-4 w-4" />
            {label}
          </button>
        ))}
      </div>

      {subTab === "provider-keys" && (
        <ProviderApiKeysPanel
          hasWriteAccess={!!hasWriteAccess}
          providers={modelsState.providers}
          apiKeysOpen={true}
          setApiKeysOpen={providerKeys.setApiKeysOpen}
          providerKeyValues={providerKeys.providerKeyValues}
          setProviderKeyValues={providerKeys.setProviderKeyValues}
          savingProviderKey={providerKeys.savingProviderKey}
          saveProviderKey={providerKeys.saveProviderKey}
          saveProviderEndpoint={providerKeys.saveProviderEndpoint}
          setError={modelsState.setError}
          load={modelsState.load}
          standalone={false}
        />
      )}

      {subTab === "catalog" && (
        <ModelCatalog
          hasWriteAccess={!!hasWriteAccess}
          providers={modelsState.providers}
          filteredCatalog={filteredCatalog}
          catalogType={catalogType}
          setCatalogType={handleCatalogTypeChange}
          catalogQuery={catalogQuery}
          setCatalogQuery={setCatalogQuery}
          catalogTierFilter={catalogTierFilter}
          setCatalogTierFilter={setCatalogTierFilter}
          catalogUseCaseFilter={catalogUseCaseFilter}
          setCatalogUseCaseFilter={setCatalogUseCaseFilter}
          catalogSort={catalogSort}
          setCatalogSort={setCatalogSort}
          pickFromCatalog={(entry) => {
            modelsState.pickFromCatalog(entry);
            setSubTab("add-model");
          }}
        />
      )}

      {subTab === "add-model" && (
        <AddModelForm
          hasWriteAccess={!!hasWriteAccess}
          providers={modelsState.providers}
          form={modelsState.form}
          setForm={modelsState.setForm}
          localProviderUrl={modelsState.localProviderUrl}
          setLocalProviderUrl={modelsState.setLocalProviderUrl}
          catalogType={catalogType}
          adding={modelsState.adding}
          onAdd={() => modelsState.handleAdd(!!hasWriteAccess)}
          providerKeyValues={providerKeys.providerKeyValues}
          saveProviderEndpoint={providerKeys.saveProviderEndpoint}
          setError={modelsState.setError}
          load={modelsState.load}
        />
      )}

      {subTab === "model-shelf" && (
        <div className="electric-card overflow-hidden">
          <div className="divide-y divide-[#1a3670]">
            {modelsState.models.length === 0 ? (
              <div className="px-4 py-8 text-center text-sm text-[#8aa8df]">
                No models configured. Add one from the Catalog or Add Model tab to get started.
              </div>
            ) : (
              modelsState.models.map((model, idx) => (
                <ModelListItem
                  key={model.id}
                  model={model}
                  index={idx}
                  totalModels={modelsState.models.length}
                  providers={modelsState.providers}
                  hasWriteAccess={!!hasWriteAccess}
                  editing={modelsState.editing === model.id}
                  deleting={modelsState.deleting === model.id}
                  editForm={modelsState.editForm[model.id]}
                  onEditFormChange={(updates) =>
                    modelsState.setEditForm((prev) => ({
                      ...prev,
                      [model.id]: { ...prev[model.id], ...updates },
                    }))
                  }
                  onMoveUp={() =>
                    modelsState.handleMove(model, "up", !!hasWriteAccess)
                  }
                  onMoveDown={() =>
                    modelsState.handleMove(model, "down", !!hasWriteAccess)
                  }
                  onStartEdit={() => modelsState.startEdit(model)}
                  onSaveEdit={() =>
                    modelsState.handleUpdate(model, !!hasWriteAccess)
                  }
                  onCancelEdit={() => modelsState.cancelEdit(model.id)}
                  onDelete={() =>
                    modelsState.handleDelete(model, !!hasWriteAccess)
                  }
                  providerKeyValues={providerKeys.providerKeyValues}
                  saveProviderEndpoint={providerKeys.saveProviderEndpoint}
                  setError={modelsState.setError}
                  load={modelsState.load}
                  savingProviderKey={providerKeys.savingProviderKey}
                />
              ))
            )}
          </div>
        </div>
      )}

    </div>
  );
}

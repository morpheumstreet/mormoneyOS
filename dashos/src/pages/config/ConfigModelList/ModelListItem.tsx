import { ChevronUp, ChevronDown, Pencil, Trash2, Check, X, Loader2 } from "lucide-react";
import { FormInput } from "@/components/ui/FormInput";
import { FormCheckbox } from "@/components/ui/FormCheckbox";
import { inputSm } from "@/lib/theme";
import type { ModelItem, ModelProvider } from "@/lib/api";
import type { EditFormState } from "./useModels";

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
}: ModelListItemProps) {
  const providerName =
    providers.find((p) => p.key === model.provider)?.displayName ?? model.provider;

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
          <div className="space-y-2">
            <div className="flex flex-wrap gap-2">
              <input
                type="text"
                value={editForm?.modelId ?? model.modelId}
                onChange={(e) =>
                  onEditFormChange({ modelId: e.target.value })
                }
                placeholder="Model ID"
                className={`flex-1 min-w-[120px] ${inputSm}`}
              />
              <input
                type="password"
                value={editForm?.apiKey ?? ""}
                onChange={(e) =>
                  onEditFormChange({ apiKey: e.target.value })
                }
                placeholder="API key (leave blank to keep)"
                className={`flex-1 min-w-[120px] ${inputSm}`}
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
                label="On"
                checked={editForm?.enabled ?? model.enabled ?? true}
                onChange={(e) =>
                  onEditFormChange({ enabled: e.target.checked })
                }
              />
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
        ) : (
          <>
            <div className="flex items-center gap-2 flex-wrap">
              <span className="font-medium text-white">{model.modelId}</span>
              <span className="text-xs text-[#6b8fcc]">
                ({providerName})
              </span>
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
            </div>
          </>
        )}
      </div>

      {!editing && hasWriteAccess && (
        <div className="flex items-center gap-1 shrink-0">
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
        </div>
      )}
    </div>
  );
}

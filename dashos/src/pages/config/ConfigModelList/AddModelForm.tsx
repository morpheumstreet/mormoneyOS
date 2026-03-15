import { Plus, Loader2 } from "lucide-react";
import { FormInput } from "@/components/ui/FormInput";
import { FormSelect } from "@/components/ui/FormSelect";
import { FormCheckbox } from "@/components/ui/FormCheckbox";
import type { ModelProvider } from "@/lib/api";
import { eligibleProviders } from "./constants";
import type { AddFormState } from "./useModels";

interface AddModelFormProps {
  hasWriteAccess: boolean;
  providers: ModelProvider[];
  form: AddFormState;
  setForm: React.Dispatch<React.SetStateAction<AddFormState>>;
  adding: boolean;
  onAdd: () => void;
}

export function AddModelForm({
  hasWriteAccess,
  providers,
  form,
  setForm,
  adding,
  onAdd,
}: AddModelFormProps) {
  if (!hasWriteAccess) return null;

  return (
    <div className="electric-card p-4">
      <div className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between xl:gap-6">
        <div className="xl:shrink-0">
          <h3 className="text-sm font-medium text-white">Add model</h3>
          <p className="text-xs text-[#8aa8df] mt-0.5">
            Pick from the catalog above, or enter manually.
          </p>
        </div>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4 xl:flex xl:flex-1 xl:flex-wrap xl:items-end xl:gap-4 xl:justify-start">
          <FormSelect
            label="Provider"
            value={form.provider}
            onChange={(e) =>
              setForm((f) => ({ ...f, provider: e.target.value }))
            }
            className="xl:w-[140px] xl:min-w-[140px]"
          >
            {eligibleProviders(providers).map((p) => (
              <option key={p.key} value={p.key}>
                {p.displayName || p.key}
              </option>
            ))}
          </FormSelect>
          <FormInput
            label="Model ID"
            value={form.modelId}
            onChange={(e) =>
              setForm((f) => ({ ...f, modelId: e.target.value }))
            }
            placeholder="e.g. llama-3.3-70b-versatile"
            className="sm:col-span-2 lg:col-span-1 xl:min-w-[200px] xl:flex-1 xl:max-w-[280px]"
          />
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
            onClick={onAdd}
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
        </div>
      </div>
    </div>
  );
}

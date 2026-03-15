import { Plus, Trash2 } from "lucide-react";
import { FormInput } from "@/components/ui/FormInput";
import { FormTextarea } from "@/components/ui/FormTextarea";
import { inputConfig } from "@/lib/theme";
import type { SoulConfig } from "@/lib/api";

interface SoulFormFieldsProps {
  config: SoulConfig;
  hasWriteAccess: boolean;
  onConfigChange: <K extends keyof SoulConfig>(key: K, value: SoulConfig[K]) => void;
  onConstraintUpdate: (idx: number, value: string) => void;
  onConstraintAdd: () => void;
  onConstraintRemove: (idx: number) => void;
}

export function SoulFormFields({
  config,
  hasWriteAccess,
  onConfigChange,
  onConstraintUpdate,
  onConstraintAdd,
  onConstraintRemove,
}: SoulFormFieldsProps) {
  const constraints = config.behavioralConstraints || [];

  return (
    <>
      <FormTextarea
        label="System prompt"
        help="Core instructions that define the agent's role and behavior."
        value={config.systemPrompt ?? ""}
        onChange={(e) => onConfigChange("systemPrompt", e.target.value)}
        disabled={!hasWriteAccess}
        rows={4}
        placeholder="You are a helpful financial assistant..."
      />

      <FormInput
        label="Personality"
        help="Traits and characteristics (e.g. helpful, analytical, curious)."
        value={config.personality ?? ""}
        onChange={(e) => onConfigChange("personality", e.target.value)}
        disabled={!hasWriteAccess}
        placeholder="helpful, analytical, curious"
      />

      <FormInput
        label="Tone"
        help="Communication style (e.g. professional, friendly, concise)."
        value={config.tone ?? ""}
        onChange={(e) => onConfigChange("tone", e.target.value)}
        disabled={!hasWriteAccess}
        placeholder="professional"
      />

      <div>
        <label className="mb-2 block text-sm font-medium text-[#8aa8df]">
          Behavioral constraints
        </label>
        <p className="mb-2 text-xs text-[#6b8fcc]">
          Rules the agent must follow (e.g. never disclose private keys).
        </p>
        <div className="space-y-2">
          {constraints.map((c, idx) => (
            <div key={idx} className="flex gap-2">
              <input
                type="text"
                value={c}
                onChange={(e) => onConstraintUpdate(idx, e.target.value)}
                disabled={!hasWriteAccess}
                placeholder="e.g. Never disclose private keys"
                className={`flex-1 ${inputConfig}`}
              />
              {hasWriteAccess && (
                <button
                  type="button"
                  onClick={() => onConstraintRemove(idx)}
                  className="p-2 rounded text-[#6b8fcc] hover:text-rose-400 hover:bg-rose-950/30"
                  aria-label="Remove constraint"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              )}
            </div>
          ))}
          {hasWriteAccess && (
            <button
              type="button"
              onClick={onConstraintAdd}
              className="flex items-center gap-2 px-3 py-2 rounded-lg border border-dashed border-[#29509c] text-[#6b8fcc] hover:text-[#9bc3ff] hover:border-[#4f83ff] transition-colors text-sm"
            >
              <Plus className="h-4 w-4" />
              Add constraint
            </button>
          )}
        </div>
      </div>
    </>
  );
}

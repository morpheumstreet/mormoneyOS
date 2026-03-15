import { FormInput } from "@/components/ui/FormInput";
import { FormSelect } from "@/components/ui/FormSelect";
import type { TreasuryPolicy } from "@/lib/api";
import { TREASURY_FIELDS } from "./constants";

const RESOURCE_MODE_OPTIONS = [
  { value: "auto", label: "Auto (credit-based)" },
  { value: "forced_on", label: "Force on (always low compute)" },
  { value: "forced_off", label: "Force off (always full compute)" },
] as const;

interface TreasuryPolicyFormProps {
  resourceMode: "auto" | "forced_on" | "forced_off";
  treasury: TreasuryPolicy;
  disabled: boolean;
  onResourceModeChange: (value: "auto" | "forced_on" | "forced_off") => void;
  onTreasuryChange: <K extends keyof TreasuryPolicy>(
    key: K,
    value: number
  ) => void;
}

export function TreasuryPolicyForm({
  resourceMode,
  treasury,
  disabled,
  onResourceModeChange,
  onTreasuryChange,
}: TreasuryPolicyFormProps) {
  return (
    <>
      <div className="electric-card p-6 space-y-4">
        <h3 className="text-sm font-medium text-white">
          Economic constraint mode
        </h3>
        <p className="text-xs text-[#6b8fcc]">
          Controls when inference uses the cheaper low-compute model based on
          credits and survival tier.
        </p>
        <FormSelect
          value={resourceMode}
          onChange={(e) =>
            onResourceModeChange(
              e.target.value as "auto" | "forced_on" | "forced_off"
            )
          }
          disabled={disabled}
        >
          {RESOURCE_MODE_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </FormSelect>
      </div>

      <div className="electric-card p-6 space-y-4">
        <h3 className="text-sm font-medium text-white">Treasury policy</h3>
        <p className="text-xs text-[#6b8fcc]">
          Financial limits for transfers and inference spend. Values in cents.
        </p>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          {TREASURY_FIELDS.map(({ key, label, defaultValue }) => (
            <FormInput
              key={key}
              type="number"
              min={0}
              label={label}
              value={treasury[key] ?? defaultValue}
              onChange={(e) =>
                onTreasuryChange(
                  key,
                  parseInt(e.target.value, 10) || 0
                )
              }
              disabled={disabled}
            />
          ))}
        </div>
      </div>
    </>
  );
}

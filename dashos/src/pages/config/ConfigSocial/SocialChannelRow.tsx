import { ChevronDown, ChevronUp, Loader2, Save } from "lucide-react";
import type { SocialChannelItem, SocialConfigField } from "@/lib/api";
import { SocialConfigFieldInput } from "./SocialConfigFieldInput";
import { formatArrayValue } from "./socialConfigUtils";

const TOGGLE_BASE_CLASS =
  "relative inline-flex h-7 w-12 shrink-0 items-center rounded-full transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-[#4f83ff] focus:ring-offset-2 focus:ring-offset-[#050d1f] disabled:opacity-50 disabled:cursor-not-allowed";
const TOGGLE_KNOB_CLASS =
  "inline-block h-5 w-5 transform rounded-full bg-white shadow transition-transform duration-200";

interface SocialChannelRowProps {
  channel: SocialChannelItem;
  expanded: boolean;
  formValues: Record<string, string | boolean>;
  toggling: boolean;
  saving: boolean;
  hasWriteAccess: boolean;
  onToggleExpand: () => void;
  onToggleEnabled: () => void;
  onSaveConfig: () => void;
  onFormValueChange: (key: string, value: string | boolean) => void;
}

export function SocialChannelRow({
  channel,
  expanded,
  formValues,
  toggling,
  saving,
  hasWriteAccess,
  onToggleExpand,
  onToggleEnabled,
  onSaveConfig,
  onFormValueChange,
}: SocialChannelRowProps) {
  const hasConfigFields =
    channel.configFields && channel.configFields.length > 0;

  const getFieldPlaceholder = (field: SocialConfigField): string | undefined => {
    if (field.type === "array") return "Comma-separated values";
    if (field.type === "password" && channel.config?.[field.key])
      return "•••••••• (leave blank to keep)";
    return undefined;
  };

  const getFieldValue = (field: SocialConfigField): string | boolean => {
    const v = formValues[field.key];
    if (v !== undefined) return v;
    if (field.type === "array")
      return formatArrayValue(channel.config?.[field.key]);
    return "";
  };

  return (
    <div className="px-4 py-3">
      <div className="flex items-center justify-between gap-4">
        <button
          type="button"
          onClick={onToggleExpand}
          className="flex min-w-0 flex-1 items-center gap-2 text-left"
        >
          {expanded ? (
            <ChevronUp className="h-4 w-4 shrink-0 text-[#7ea5eb]" />
          ) : (
            <ChevronDown className="h-4 w-4 shrink-0 text-[#7ea5eb]" />
          )}
          <p className="font-medium text-white">
            {channel.displayName || channel.name}
          </p>
          {!channel.ready && channel.enabled && (
            <span className="text-xs text-amber-400/90">Not configured</span>
          )}
        </button>
        <button
          type="button"
          onClick={onToggleEnabled}
          disabled={!hasWriteAccess || !!toggling}
          className={`${TOGGLE_BASE_CLASS} ${
            channel.enabled ? "bg-[#2f8fff]/60" : "bg-[#1a3670]"
          }`}
          role="switch"
          aria-checked={channel.enabled}
        >
          <span
            className={`${TOGGLE_KNOB_CLASS} ${
              channel.enabled ? "translate-x-6" : "translate-x-1"
            }`}
          />
          {toggling && (
            <span className="absolute inset-0 flex items-center justify-center">
              <Loader2 className="h-4 w-4 animate-spin text-[#9bc3ff]" />
            </span>
          )}
        </button>
      </div>

      {expanded && hasConfigFields && (
        <div className="mt-4 space-y-3 border-t border-[#1a3670] pt-4">
          {channel.configFields!.map((field) => (
            <div key={field.key}>
              <SocialConfigFieldInput
                field={field}
                value={getFieldValue(field)}
                placeholder={getFieldPlaceholder(field)}
                onChange={(v) => onFormValueChange(field.key, v)}
                disabled={!hasWriteAccess}
              />
            </div>
          ))}
          <button
            type="button"
            onClick={onSaveConfig}
            disabled={!hasWriteAccess || !!saving}
            className="electric-button flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium disabled:opacity-50"
          >
            {saving ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Save className="h-4 w-4" />
            )}
            {saving ? "Saving…" : "Save config"}
          </button>
        </div>
      )}
    </div>
  );
}

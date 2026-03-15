import { ChevronDown, ChevronUp, Save, RefreshCw, Loader2 } from "lucide-react";
import type { TunnelProviderField, TunnelProviderSchema } from "@/lib/api";
import { PROVIDER_LABELS, PROVIDERS_WITH_RESTART } from "./constants";
import { SchemaFormField } from "./SchemaFormField";

interface TunnelProviderRowProps {
  name: string;
  schema: TunnelProviderSchema | undefined;
  providerConfig: Record<string, unknown> | undefined;
  expanded: boolean;
  formValues: Record<string, string | boolean>;
  saving: boolean;
  restarting: boolean;
  toggling: boolean;
  hasWriteAccess: boolean;
  needsConfig: boolean;
  isConfigured: (name: string, field: TunnelProviderField) => boolean;
  onToggleExpand: () => void;
  onToggleEnabled: () => void;
  onRestart: () => void;
  onSaveConfig: () => void;
  onFormValueChange: (key: string, value: string | boolean) => void;
}

export function TunnelProviderRow({
  name,
  schema,
  providerConfig,
  expanded,
  formValues,
  saving,
  restarting,
  toggling,
  hasWriteAccess,
  needsConfig,
  isConfigured,
  onToggleExpand,
  onToggleEnabled,
  onRestart,
  onSaveConfig,
  onFormValueChange,
}: TunnelProviderRowProps) {
  const enabled = !!providerConfig?.enabled;
  const showRestart = PROVIDERS_WITH_RESTART.includes(name);

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
            {PROVIDER_LABELS[name] ?? name}
          </p>
          {needsConfig && (
            <span className="text-xs text-amber-400/90">API key required</span>
          )}
        </button>
        <button
          type="button"
          onClick={onToggleEnabled}
          disabled={!hasWriteAccess || !!toggling}
          className={`
            relative inline-flex h-7 w-12 shrink-0 items-center rounded-full
            transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-[#4f83ff] focus:ring-offset-2 focus:ring-offset-[#050d1f]
            disabled:opacity-50 disabled:cursor-not-allowed
            ${enabled ? "bg-[#2f8fff]/60" : "bg-[#1a3670]"}
          `}
          role="switch"
          aria-checked={enabled}
        >
          <span
            className={`
              inline-block h-5 w-5 transform rounded-full bg-white shadow
              transition-transform duration-200
              ${enabled ? "translate-x-6" : "translate-x-1"}
            `}
          />
          {toggling && (
            <span className="absolute inset-0 flex items-center justify-center">
              <Loader2 className="h-4 w-4 animate-spin text-[#9bc3ff]" />
            </span>
          )}
        </button>
        {showRestart && (
          <button
            type="button"
            onClick={onRestart}
            disabled={!hasWriteAccess || !!restarting || needsConfig}
            title="Reload provider from config (requires API key in automaton.json)"
            className="electric-button flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium disabled:opacity-50"
          >
            {restarting ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <RefreshCw className="h-3.5 w-3.5" />
            )}
            Restart
          </button>
        )}
      </div>

      {expanded && schema?.fields && schema.fields.length > 0 && (
        <div className="mt-4 space-y-3 border-t border-[#1a3670] pt-4">
          {schema.fields.map((field) => (
            <SchemaFormField
              key={field.name}
              field={field}
              value={formValues[field.name] ?? (field.type === "boolean" ? false : "")}
              onChange={(v) => onFormValueChange(field.name, v)}
              disabled={!hasWriteAccess}
              isConfigured={isConfigured(name, field)}
            />
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

      {expanded && (!schema?.fields || schema.fields.length === 0) && (
        <div className="mt-4 border-t border-[#1a3670] pt-4 text-sm text-[#8aa8df]">
          No configuration required. Provider is ready when enabled.
        </div>
      )}
    </div>
  );
}

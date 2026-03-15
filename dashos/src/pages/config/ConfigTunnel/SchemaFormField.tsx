import { FormInput } from "@/components/ui/FormInput";
import { FormCheckbox } from "@/components/ui/FormCheckbox";
import type { TunnelProviderField } from "@/lib/api";

interface SchemaFormFieldProps {
  field: TunnelProviderField;
  value: string | boolean;
  onChange: (value: string | boolean) => void;
  disabled?: boolean;
  isConfigured?: boolean;
}

export function SchemaFormField({
  field,
  value,
  onChange,
  disabled = false,
  isConfigured = false,
}: SchemaFormFieldProps) {
  if (field.type === "boolean") {
    return (
      <FormCheckbox
        label="Enable"
        checked={(value as boolean) ?? false}
        onChange={(e) => onChange(e.target.checked)}
        disabled={disabled}
      />
    );
  }

  const placeholder =
    field.type === "password" && isConfigured
      ? "•••••••• (leave blank to keep)"
      : field.type === "password"
        ? "Enter API key or use ${ENV_VAR}"
        : undefined;

  return (
    <FormInput
      label={field.label ?? field.name}
      help={field.help}
      required={field.required}
      type={field.type === "password" ? "password" : "text"}
      value={(value as string) ?? ""}
      onChange={(e) => onChange(e.target.value)}
      disabled={disabled}
      placeholder={placeholder}
    />
  );
}

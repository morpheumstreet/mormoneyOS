import { FormInput } from "@/components/ui/FormInput";
import { FormCheckbox } from "@/components/ui/FormCheckbox";
import type { SocialConfigField } from "@/lib/api";

interface SocialConfigFieldInputProps {
  field: SocialConfigField;
  value: string | boolean;
  placeholder?: string;
  onChange: (value: string | boolean) => void;
  disabled?: boolean;
}

/** Renders a single social config field based on its type. */
export function SocialConfigFieldInput({
  field,
  value,
  placeholder,
  onChange,
  disabled = false,
}: SocialConfigFieldInputProps) {
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

  return (
    <FormInput
      label={field.label}
      help={field.description}
      required={field.required}
      type={field.type === "password" ? "password" : "text"}
      value={(value as string) ?? ""}
      onChange={(e) => onChange(e.target.value)}
      disabled={disabled}
      placeholder={placeholder}
    />
  );
}

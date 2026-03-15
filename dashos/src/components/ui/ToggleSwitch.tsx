import { Loader2 } from "lucide-react";

interface ToggleSwitchProps {
  checked: boolean;
  disabled?: boolean;
  loading?: boolean;
  label: string;
  onChange: () => void;
}

const SWITCH_CLASSES = `
  relative inline-flex h-7 w-12 shrink-0 items-center rounded-full
  transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-[#4f83ff] focus:ring-offset-2 focus:ring-offset-[#050d1f]
  disabled:opacity-50 disabled:cursor-not-allowed
`;

const KNOB_CLASSES = `
  inline-block h-5 w-5 transform rounded-full bg-white shadow
  transition-transform duration-200
`;

export function ToggleSwitch({
  checked,
  disabled = false,
  loading = false,
  label,
  onChange,
}: ToggleSwitchProps) {
  return (
    <button
      type="button"
      onClick={onChange}
      disabled={disabled || loading}
      className={`${SWITCH_CLASSES} ${checked ? "bg-[#2f8fff]/60" : "bg-[#1a3670]"}`}
      role="switch"
      aria-checked={checked}
      aria-label={label}
    >
      <span
        className={`${KNOB_CLASSES} ${checked ? "translate-x-6" : "translate-x-1"}`}
      />
      {loading && (
        <span className="absolute inset-0 flex items-center justify-center">
          <Loader2 className="h-4 w-4 animate-spin text-[#9bc3ff]" />
        </span>
      )}
    </button>
  );
}

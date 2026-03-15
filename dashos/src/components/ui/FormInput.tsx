import { inputSm, inputMd } from "@/lib/theme";

interface FormInputProps
  extends Omit<
    React.InputHTMLAttributes<HTMLInputElement>,
    "className"
  > {
  label?: string;
  help?: string;
  required?: boolean;
  size?: "sm" | "md";
  className?: string;
}

export function FormInput({
  label,
  help,
  required,
  size = "md",
  className = "",
  ...props
}: FormInputProps) {
  const inputClass = size === "sm" ? inputSm : inputMd;
  return (
    <div className={className}>
      {label && (
        <label className="mb-1 block text-xs font-medium text-[#8aa8df]">
          {label}
          {required && <span className="ml-1 text-amber-400">*</span>}
        </label>
      )}
      {help && (
        <p className="mb-1 text-xs text-[#6b8fcc]">{help}</p>
      )}
      <input
        {...props}
        className={`w-full ${inputClass}`}
      />
    </div>
  );
}

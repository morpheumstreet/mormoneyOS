import { inputMd } from "@/lib/theme";

interface FormSelectProps
  extends Omit<
    React.SelectHTMLAttributes<HTMLSelectElement>,
    "className"
  > {
  label?: string;
  children: React.ReactNode;
  className?: string;
}

export function FormSelect({
  label,
  children,
  className = "",
  ...props
}: FormSelectProps) {
  return (
    <div className={className}>
      {label && (
        <label className="mb-1 block text-xs font-medium text-[#8aa8df]">
          {label}
        </label>
      )}
      <select {...props} className={`w-full ${inputMd}`}>
        {children}
      </select>
    </div>
  );
}

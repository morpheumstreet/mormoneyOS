import { inputConfig } from "@/lib/theme";

interface FormTextareaProps
  extends Omit<
    React.TextareaHTMLAttributes<HTMLTextAreaElement>,
    "className"
  > {
  label?: string;
  help?: string;
  className?: string;
}

export function FormTextarea({
  label,
  help,
  className = "",
  ...props
}: FormTextareaProps) {
  return (
    <div className={className}>
      {label && (
        <label className="mb-1 block text-xs font-medium text-[#8aa8df]">
          {label}
        </label>
      )}
      {help && (
        <p className="mb-1 text-xs text-[#6b8fcc]">{help}</p>
      )}
      <textarea
        {...props}
        className={`${inputConfig} resize-y`}
      />
    </div>
  );
}

interface FormCheckboxProps
  extends Omit<
    React.InputHTMLAttributes<HTMLInputElement>,
    "type" | "className"
  > {
  label: string;
  className?: string;
}

export function FormCheckbox({
  label,
  className = "",
  ...props
}: FormCheckboxProps) {
  return (
    <label
      className={`flex items-center gap-2 cursor-pointer ${className}`}
    >
      <input
        type="checkbox"
        {...props}
        className="rounded border-[#29509c] bg-[#071228]/90 text-[#4f83ff] focus:ring-[#4f83ff]"
      />
      <span className="text-xs text-[#8aa8df]">{label}</span>
    </label>
  );
}

import { AlertTriangle, CheckCircle } from "lucide-react";

type Variant = "success" | "error";

const styles: Record<Variant, string> = {
  success:
    "electric-card p-3 border-emerald-500/30 bg-emerald-950/20 flex items-center gap-2 text-emerald-300",
  error:
    "electric-card p-3 border-rose-500/30 bg-rose-950/20 flex items-center gap-2 text-rose-300",
};

const icons = {
  success: CheckCircle,
  error: AlertTriangle,
};

const iconStyles: Record<Variant, string> = {
  success: "text-emerald-400",
  error: "text-rose-400",
};

interface AlertMessageProps {
  variant: Variant;
  message: string;
  className?: string;
}

export function AlertMessage({ variant, message, className = "" }: AlertMessageProps) {
  const Icon = icons[variant];
  return (
    <div className={`${styles[variant]} ${className}`}>
      <Icon className={`h-4 w-4 flex-shrink-0 ${iconStyles[variant]}`} />
      <span className="text-sm">{message}</span>
    </div>
  );
}

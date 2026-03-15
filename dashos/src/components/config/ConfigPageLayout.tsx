import { type LucideIcon } from "lucide-react";
import { AlertTriangle, CheckCircle } from "lucide-react";

interface ConfigPageLayoutProps {
  icon: LucideIcon;
  title: string;
  description: string;
  hasWriteAccess: boolean;
  writeAccessMessage?: string;
  error: string | null;
  loading: boolean;
  success?: string | null;
  headerActions?: React.ReactNode;
  children: React.ReactNode;
}

export function ConfigPageLayout({
  icon: Icon,
  title,
  description,
  hasWriteAccess,
  writeAccessMessage = "Connect your wallet and sign to configure.",
  error,
  success,
  loading,
  headerActions,
  children,
}: ConfigPageLayoutProps) {
  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="electric-loader h-12 w-12 rounded-full" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className={`flex items-center ${headerActions ? "justify-between" : "gap-3"}`}>
        <div className="flex items-center gap-3">
          <div className="electric-icon h-10 w-10 rounded-xl flex items-center justify-center">
            <Icon className="h-5 w-5 text-[#9bc3ff]" />
          </div>
          <div>
            <h2 className="text-lg font-semibold text-white">{title}</h2>
            <p className="text-sm text-[#8aa8df]">{description}</p>
          </div>
        </div>
        {headerActions}
      </div>

      {!hasWriteAccess && (
        <div className="electric-card p-4 border-amber-500/30 bg-amber-950/20">
          <div className="flex items-start gap-3">
            <AlertTriangle className="h-5 w-5 text-amber-400 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-amber-200">Write access required</p>
              <p className="text-sm text-amber-300/80 mt-1">{writeAccessMessage}</p>
            </div>
          </div>
        </div>
      )}

      {success && (
        <div className="electric-card p-3 border-emerald-500/30 bg-emerald-950/20 flex items-center gap-2">
          <CheckCircle className="h-4 w-4 text-emerald-400 flex-shrink-0" />
          <span className="text-sm text-emerald-300">{success}</span>
        </div>
      )}

      {error && (
        <div className="electric-card p-3 border-rose-500/30 bg-rose-950/20 flex items-center gap-2">
          <AlertTriangle className="h-4 w-4 text-rose-400 flex-shrink-0" />
          <span className="text-sm text-rose-300">{error}</span>
        </div>
      )}

      {children}
    </div>
  );
}

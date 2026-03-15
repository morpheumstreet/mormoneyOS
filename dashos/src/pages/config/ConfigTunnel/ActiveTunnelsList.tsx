import { ExternalLink } from "lucide-react";
import type { TunnelItem } from "./useTunnelProviders";

interface ActiveTunnelsListProps {
  tunnels: TunnelItem[];
}

export function ActiveTunnelsList({ tunnels }: ActiveTunnelsListProps) {
  if (tunnels.length === 0) return null;

  return (
    <div className="electric-card p-4">
      <h3 className="text-sm font-medium text-white mb-3">Active tunnels</h3>
      <div className="space-y-2">
        {tunnels.map((t) => (
          <div
            key={`${t.port}-${t.provider}`}
            className="flex items-center justify-between gap-4 rounded-lg border border-[#1a3670] bg-[#071228]/50 px-3 py-2"
          >
            <div className="min-w-0">
              <p className="text-sm font-medium text-white">
                Port {t.port} ({t.provider})
              </p>
              <a
                href={t.public_url}
                target="_blank"
                rel="noopener noreferrer"
                className="text-xs text-[#7ea5eb] hover:underline flex items-center gap-1 truncate"
              >
                {t.public_url}
                <ExternalLink className="h-3 w-3 flex-shrink-0" />
              </a>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

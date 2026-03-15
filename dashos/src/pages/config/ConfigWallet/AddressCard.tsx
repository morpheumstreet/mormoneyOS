import { Copy, QrCode } from "lucide-react";
import { truncateAddress } from "@/lib/format";
import type { ChainAddress } from "./useWalletConfig";

interface AddressCardProps {
  item: ChainAddress;
  onCopy: (addr: string) => void;
  onShowQr?: (addr: string, label: string) => void;
}

export function AddressCard({ item, onCopy, onShowQr }: AddressCardProps) {
  const displayAddr = item.address || (item.error ? `Error: ${item.error}` : "—");
  const canCopy = !!item.address;

  return (
    <div
      className={`electric-card p-4 transition-all ${
        item.isMorpheum
          ? "border-[#00d4aa]/50 shadow-[0_0_20px_-8px_rgba(0,212,170,0.4)]"
          : ""
      }`}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-[#9bb7eb]">{item.name}</span>
            {item.isMorpheum && (
              <span className="rounded bg-[#00d4aa]/20 px-1.5 py-0.5 text-[10px] font-medium text-[#00d4aa]">
                Hybrid Post-Quantum
              </span>
            )}
          </div>
          <p className="mt-1 font-mono text-sm text-white break-all">
            {canCopy ? truncateAddress(item.address, 8, 6) : displayAddr}
          </p>
        </div>
        <div className="flex shrink-0 gap-1">
          {canCopy && (
            <>
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  onCopy(item.address);
                }}
                className="rounded p-1.5 text-[#8aa8df] hover:bg-[#1a3670]/60 hover:text-white transition-colors"
                title="Copy"
              >
                <Copy className="h-4 w-4" />
              </button>
              {onShowQr && (
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    onShowQr(item.address, item.name);
                  }}
                  className="rounded p-1.5 text-[#8aa8df] hover:bg-[#1a3670]/60 hover:text-white transition-colors"
                  title="QR code"
                >
                  <QrCode className="h-4 w-4" />
                </button>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
}

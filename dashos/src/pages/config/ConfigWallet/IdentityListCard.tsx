import { useState, useEffect } from "react";
import { ChevronRight } from "lucide-react";
import { truncateAddress } from "@/lib/format";

interface IdentityListCardProps {
  index: number;
  label: string;
  isCurrent: boolean;
  defaultChain?: string;
  chainName: (caip2: string) => string;
  getPrimaryAddress: (chain: string, idx: number) => Promise<string>;
  onClick: () => void;
}

export function IdentityListCard({
  index,
  label,
  isCurrent,
  defaultChain,
  chainName,
  getPrimaryAddress,
  onClick,
}: IdentityListCardProps) {
  const [primaryAddr, setPrimaryAddr] = useState<string | null>(null);
  const chain = defaultChain || "eip155:8453";

  useEffect(() => {
    getPrimaryAddress(chain, index).then(setPrimaryAddr);
  }, [chain, index, getPrimaryAddress]);

  const displayName = label?.trim() || `Account #${index}`;

  return (
    <div
      className="electric-card overflow-hidden flex items-center gap-4 p-4 cursor-pointer hover:bg-[#0b2f80]/30 transition-colors"
      onClick={onClick}
    >
      <div
        className="w-12 h-12 rounded-full shrink-0"
        style={{ background: "linear-gradient(135deg, #3a6de0 0%, #00d4aa 100%)" }}
      />
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <p className="font-medium text-white">{displayName}</p>
          {isCurrent && (
            <span className="rounded bg-[#00d4aa]/20 px-1.5 py-0.5 text-[10px] font-medium text-[#00d4aa]">
              Active
            </span>
          )}
        </div>
        <p className="text-sm text-[#6b8fcc]">
          {chainName(chain)} · {primaryAddr ? truncateAddress(primaryAddr, 6, 4) : "…"}
        </p>
      </div>
      <ChevronRight className="h-5 w-5 text-[#6b8fcc] shrink-0" />
    </div>
  );
}

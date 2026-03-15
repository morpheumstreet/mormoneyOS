import { Copy } from "lucide-react";
import { truncateAddress } from "@/lib/format";

interface CopyWithTruncateProps {
  address: string;
  head?: number;
  tail?: number;
  onCopy: (addr: string) => void;
  className?: string;
}

export function CopyWithTruncate({
  address,
  head = 6,
  tail = 4,
  onCopy,
  className = "",
}: CopyWithTruncateProps) {
  return (
    <span className={className}>
      {truncateAddress(address, head, tail)}
      <button
        type="button"
        onClick={() => onCopy(address)}
        className="ml-2 inline p-1 rounded text-[#8aa8df] hover:bg-[#1a3670]"
      >
        <Copy className="h-3.5 w-3.5" />
      </button>
    </span>
  );
}

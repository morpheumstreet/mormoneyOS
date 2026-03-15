import { ChevronRight } from "lucide-react";
import { BackButton } from "@/components/ui/BackButton";
import { AddressCard } from "../AddressCard";
import type { ChainAddress } from "../useWalletConfig";

interface CreatePreviewScreenProps {
  createIndex: number;
  previewAddresses: ChainAddress[];
  onCopy: (addr: string) => void;
  onShowQr: (addr: string, label: string) => void;
  onContinue: () => void;
  onBack: () => void;
}

export function CreatePreviewScreen({
  createIndex,
  previewAddresses,
  onCopy,
  onShowQr,
  onContinue,
  onBack,
}: CreatePreviewScreenProps) {
  return (
    <div className="motion-rise space-y-6">
      <div className="flex items-center gap-4">
        <BackButton onClick={onBack} />
        <span className="text-sm text-[#6b8fcc]">
          Your new multi-chain identity (index {createIndex})
        </span>
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {previewAddresses.map((item) => (
          <AddressCard
            key={item.caip2}
            item={item}
            onCopy={onCopy}
            onShowQr={onShowQr}
          />
        ))}
      </div>
      <div className="flex justify-end">
        <button
          type="button"
          onClick={onContinue}
          className="electric-button flex items-center gap-2 px-5 py-2.5 rounded-lg"
        >
          Looks good!
          <ChevronRight className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
}

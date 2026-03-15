import { CheckCircle } from "lucide-react";
import { CopyWithTruncate } from "@/components/ui/CopyWithTruncate";
import { chainName } from "../constants";

interface CreateSuccessScreenProps {
  createIndex: number;
  identityLabel: string;
  selectedChain: string;
  primaryAddress: string;
  morpheumAddr: string;
  onCopy: (addr: string) => void;
  onContinue: () => void;
  onCreateAnother: () => void;
}

export function CreateSuccessScreen({
  createIndex,
  identityLabel,
  selectedChain,
  primaryAddress,
  morpheumAddr,
  onCopy,
  onContinue,
  onCreateAnother,
}: CreateSuccessScreenProps) {
  return (
    <div className="motion-rise text-center py-8">
      <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-emerald-500/20 border border-emerald-500/40 mb-4">
        <CheckCircle className="h-8 w-8 text-emerald-400" />
      </div>
      <h2 className="text-2xl font-bold text-white mb-2">Identity Created!</h2>
      <p className="text-[#8aa8df] mb-6">
        Account #{createIndex} – &quot;{identityLabel}&quot;
      </p>
      <div className="electric-card p-6 max-w-md mx-auto text-left space-y-2">
        <p><span className="text-[#6b8fcc]">Name:</span> {identityLabel}</p>
        <p><span className="text-[#6b8fcc]">Index:</span> {createIndex}</p>
        <p><span className="text-[#6b8fcc]">Default chain:</span> {chainName(selectedChain)}</p>
        <p>
          <span className="text-[#6b8fcc]">Primary address:</span>{" "}
          <CopyWithTruncate address={primaryAddress} onCopy={onCopy} />
        </p>
        {morpheumAddr && (
          <p>
            <span className="text-[#6b8fcc]">Morpheum address:</span>{" "}
            <CopyWithTruncate
              address={morpheumAddr}
              head={8}
              tail={6}
              onCopy={onCopy}
              className="text-[#00d4aa]"
            />
          </p>
        )}
      </div>
      <div className="flex flex-col sm:flex-row gap-4 justify-center mt-8">
        <button
          type="button"
          onClick={onContinue}
          className="electric-button px-6 py-2.5 rounded-lg"
        >
          Continue to Dashboard
        </button>
        <button
          type="button"
          onClick={onCreateAnother}
          className="border border-[#3a6de0] text-[#9bb7eb] hover:bg-[#0b2f80]/40 px-6 py-2.5 rounded-lg"
        >
          Create Another Identity
        </button>
      </div>
    </div>
  );
}

import { BackButton } from "@/components/ui/BackButton";
import { IdentityListCard } from "../IdentityListCard";
import { chainName } from "../constants";

const FOOTNOTE =
  "Private keys & mnemonic are securely managed — never shown here.";

interface ManageScreenProps {
  identityLabels: Record<string, string>;
  currentIndex: number | undefined;
  defaultChain?: string;
  getPrimaryAddress: (chain: string, idx: number) => Promise<string>;
  onDetail: (index: number) => void;
  onBack: () => void;
}

export function ManageScreen({
  identityLabels,
  currentIndex,
  defaultChain,
  getPrimaryAddress,
  onDetail,
  onBack,
}: ManageScreenProps) {
  const indices = new Set<number>(
    [...Object.keys(identityLabels).map(Number), currentIndex ?? -1].filter((n) => n >= 0)
  );
  const sorted = [...indices].sort((a, b) => a - b);

  return (
    <div className="motion-rise space-y-6">
      <div className="flex items-center gap-4">
        <BackButton onClick={onBack} />
      </div>
      <div className="space-y-3">
        {sorted.length === 0 ? (
          <div className="electric-card p-6 text-center text-[#6b8fcc]">
            <p className="text-sm">No identities yet. Create one to get started.</p>
          </div>
        ) : (
          sorted.map((idx) => (
            <IdentityListCard
              key={idx}
              index={idx}
              label={identityLabels[String(idx)]}
              isCurrent={idx === currentIndex}
              defaultChain={defaultChain}
              chainName={chainName}
              getPrimaryAddress={getPrimaryAddress}
              onClick={() => onDetail(idx)}
            />
          ))
        )}
      </div>
      <p className="text-xs text-[#6b8fcc]">{FOOTNOTE}</p>
    </div>
  );
}

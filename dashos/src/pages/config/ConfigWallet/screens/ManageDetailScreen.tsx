import { BackButton } from "@/components/ui/BackButton";
import type { ChainAddress } from "../useWalletConfig";
import { AlertMessage } from "@/components/ui/AlertMessage";
import { InlineEdit } from "@/components/ui/InlineEdit";
import { ManageDetailAddresses } from "../ManageDetailAddresses";

interface ManageDetailScreenProps {
  detailIndex: number;
  currentIndex: number | undefined;
  labelEdit: string;
  success: string | null;
  error: string | null;
  hasWriteAccess: boolean;
  savingLabel: boolean;
  onSaveLabel: (newLabel: string) => Promise<void>;
  onSwitchIdentity: () => Promise<void>;
  deriveAddressesForIndex: (i: number) => Promise<ChainAddress[]>;
  onCopy: (addr: string) => void;
  onShowQr: (addr: string, label: string) => void;
  onBack: () => void;
}

export function ManageDetailScreen({
  detailIndex,
  currentIndex,
  labelEdit,
  success,
  error,
  hasWriteAccess,
  savingLabel,
  onSaveLabel,
  onSwitchIdentity,
  deriveAddressesForIndex,
  onCopy,
  onShowQr,
  onBack,
}: ManageDetailScreenProps) {
  const isCurrent = detailIndex === currentIndex;

  return (
    <div className="motion-rise py-8">
      <div className="flex justify-start mb-6">
        <BackButton onClick={onBack} />
      </div>

      <div className="text-center">
        {success && (
          <AlertMessage variant="success" message={success} className="max-w-md mx-auto mb-4" />
        )}
        {error && (
          <AlertMessage variant="error" message={error} className="max-w-md mx-auto mb-4" />
        )}

        <div className="flex flex-col items-center mb-6">
          <div
            className="flex-shrink-0 w-16 h-16 rounded-full mb-4"
            style={{ background: "linear-gradient(135deg, #3a6de0 0%, #00d4aa 100%)" }}
          />
          <InlineEdit
            value={labelEdit?.trim() || ""}
            onSave={onSaveLabel}
            disabled={!hasWriteAccess || savingLabel}
            placeholder={`Account #${detailIndex}`}
            as="h2"
            className="text-2xl font-bold text-white mb-2 text-center"
            inputClassName="text-2xl font-bold text-white text-center"
          />
          <p className="text-[#8aa8df]">Identity #{detailIndex}</p>
        </div>

        {!isCurrent && (
          <div className="max-w-md mx-auto mb-6">
            <button
              type="button"
              onClick={onSwitchIdentity}
              disabled={!hasWriteAccess}
              className="electric-button w-full py-2.5 rounded-lg text-sm font-medium disabled:opacity-50"
            >
              Use this identity
            </button>
            <p className="mt-1 text-xs text-[#6b8fcc] text-center">
              Switch the agent to use Account #{detailIndex} for signing and provisioning.
            </p>
          </div>
        )}

        <div className="max-w-2xl mx-auto mt-6 text-left">
          <h3 className="text-sm font-medium text-[#9bb7eb] mb-3">Addresses by chain</h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <ManageDetailAddresses
              index={detailIndex}
              deriveAddressesForIndex={deriveAddressesForIndex}
              onCopy={onCopy}
              onShowQr={onShowQr}
            />
          </div>
        </div>

        <div className="flex flex-col sm:flex-row gap-4 justify-center mt-8">
          <button
            type="button"
            onClick={onBack}
            className="electric-button px-6 py-2.5 rounded-lg"
          >
            Back to identities
          </button>
          {!isCurrent && hasWriteAccess && (
            <button
              type="button"
              onClick={onSwitchIdentity}
              className="border border-[#3a6de0] text-[#9bb7eb] hover:bg-[#0b2f80]/40 px-6 py-2.5 rounded-lg"
            >
              Use this identity
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

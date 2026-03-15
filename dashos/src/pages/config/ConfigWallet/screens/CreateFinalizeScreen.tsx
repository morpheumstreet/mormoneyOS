import { BackButton } from "@/components/ui/BackButton";
import { AlertMessage } from "@/components/ui/AlertMessage";
import { SUPPORTED_CHAINS } from "../constants";

interface CreateFinalizeScreenProps {
  error: string | null;
  identityLabel: string;
  setIdentityLabel: (v: string) => void;
  understoodCheck: boolean;
  setUnderstoodCheck: (v: boolean) => void;
  selectedChain: string;
  setSelectedChain: (v: string) => void;
  onCreate: () => void;
  creating: boolean;
  hasWriteAccess: boolean;
  onBack: () => void;
}

export function CreateFinalizeScreen({
  error,
  identityLabel,
  setIdentityLabel,
  understoodCheck,
  setUnderstoodCheck,
  selectedChain,
  setSelectedChain,
  onCreate,
  creating,
  hasWriteAccess,
  onBack,
}: CreateFinalizeScreenProps) {
  return (
    <div className="motion-rise space-y-6">
      <div className="flex items-center gap-4">
        <BackButton onClick={onBack} />
        <span className="text-sm text-[#6b8fcc]">Almost there · Step 2/3</span>
      </div>

      {error && <AlertMessage variant="error" message={error} />}

      <div className="electric-card p-6 space-y-6">
        <div>
          <h3 className="text-sm font-medium text-[#9bb7eb] mb-2">Primary chain</h3>
          <p className="text-xs text-[#6b8fcc] mb-3">
            Used for provisioning, heartbeats, and when no chain is specified.
          </p>
          <div className="flex flex-wrap gap-2">
            {SUPPORTED_CHAINS.map((ch) => (
              <button
                key={ch.caip2}
                type="button"
                onClick={() => setSelectedChain(ch.caip2)}
                className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-all ${
                  selectedChain === ch.caip2
                    ? ch.isMorpheum
                      ? "bg-[#00d4aa]/30 text-[#00d4aa] border border-[#00d4aa]/50"
                      : "bg-[#4f83ff]/30 text-[#9bc3ff] border border-[#4f83ff]/50"
                    : "border border-[#29509c] text-[#8aa8df] hover:bg-[#07132f]"
                }`}
              >
                {ch.name}
              </button>
            ))}
          </div>
        </div>

        <div>
          <h3 className="text-sm font-medium text-[#9bb7eb] mb-2">Give it a name</h3>
          <input
            type="text"
            value={identityLabel}
            onChange={(e) => setIdentityLabel(e.target.value)}
            placeholder="My Trading Agent #3"
            className="w-full px-4 py-3 rounded-lg border border-[#29509c] bg-[#071228]/90 text-white placeholder:text-[#6b8fcc] focus:border-[#4f83ff] focus:outline-none"
            maxLength={64}
          />
          <p className="mt-1 text-xs text-[#6b8fcc]">{identityLabel.length}/64</p>
        </div>

        <label className="flex items-start gap-3 cursor-pointer">
          <input
            type="checkbox"
            checked={understoodCheck}
            onChange={(e) => setUnderstoodCheck(e.target.checked)}
            className="mt-1 rounded border-[#29509c]"
          />
          <span className="text-sm text-[#8aa8df]">
            I understand private keys are never shown and are managed securely by the backend.
          </span>
        </label>

        <button
          type="button"
          onClick={onCreate}
          disabled={creating || !identityLabel.trim() || !understoodCheck || !hasWriteAccess}
          className="electric-button w-full py-3 rounded-lg font-medium disabled:opacity-50"
        >
          {creating ? "Creating…" : "Create & Save Identity"}
        </button>
      </div>
    </div>
  );
}

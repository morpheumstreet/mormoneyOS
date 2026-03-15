import { Plus, List } from "lucide-react";
import { BackButton } from "@/components/ui/BackButton";

const FOOTNOTE =
  "Private keys & mnemonic are securely managed — never shown here.";

interface WelcomeScreenProps {
  onCreate: () => void;
  onManage: () => void;
}

export function WelcomeScreen({ onCreate, onManage }: WelcomeScreenProps) {
  return (
    <div className="motion-rise">
      <div className="text-center max-w-2xl mx-auto py-8">
        <h2 className="text-2xl font-bold text-white mb-2">
          Give your Automaton its next identity
        </h2>
        <p className="text-[#8aa8df] text-sm mb-8">
          One secure seed → many chains. Powered by Morpheum standards.
        </p>
        <div className="flex flex-col sm:flex-row gap-4 justify-center">
          <button
            type="button"
            onClick={onCreate}
            className="electric-button flex items-center justify-center gap-2 px-6 py-4 rounded-xl text-base font-medium"
          >
            <Plus className="h-5 w-5" />
            Create New Identity
          </button>
          <button
            type="button"
            onClick={onManage}
            className="flex items-center justify-center gap-2 px-6 py-4 rounded-xl text-base font-medium border border-[#3a6de0] text-[#9bb7eb] hover:bg-[#0b2f80]/40 hover:border-[#4f83ff] transition-all"
          >
            <List className="h-5 w-5" />
            Manage Existing Identities
          </button>
        </div>
        <p className="mt-8 text-xs text-[#6b8fcc]">{FOOTNOTE}</p>
      </div>
    </div>
  );
}

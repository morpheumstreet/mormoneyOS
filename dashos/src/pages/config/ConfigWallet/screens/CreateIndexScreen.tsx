import { ChevronRight } from "lucide-react";
import { BackButton } from "@/components/ui/BackButton";

interface CreateIndexScreenProps {
  createIndex: number;
  setCreateIndex: (n: number) => void;
  currentIndex: number | null;
  onPreview: () => void;
  onBack: () => void;
}

export function CreateIndexScreen({
  createIndex,
  setCreateIndex,
  currentIndex,
  onPreview,
  onBack,
}: CreateIndexScreenProps) {
  const decrement = () =>
    setCreateIndex(Math.max(1, (currentIndex ?? 0) + 1, createIndex - 1));
  const random = () => setCreateIndex(10000 + Math.floor(Math.random() * 90000));

  return (
    <div className="motion-rise space-y-6">
      <div className="flex items-center gap-4">
        <BackButton onClick={onBack} />
        <span className="text-sm text-[#6b8fcc]">Create New Identity · Step 1/3</span>
      </div>
      <div className="electric-card p-6">
        <p className="text-sm text-[#9bb7eb] mb-2">
          Next available account index: <strong className="text-white">{createIndex}</strong>
        </p>
        <p className="text-xs text-[#6b8fcc] mb-4">
          {currentIndex != null && `(last used was ${currentIndex})`}
        </p>
        <div className="flex items-center gap-4">
          <button
            type="button"
            onClick={decrement}
            className="electric-button px-4 py-2 rounded-lg"
          >
            −
          </button>
          <span className="text-2xl font-mono font-bold text-white min-w-[3rem] text-center">
            {createIndex}
          </span>
          <button
            type="button"
            onClick={() => setCreateIndex(createIndex + 1)}
            className="electric-button px-4 py-2 rounded-lg"
          >
            +
          </button>
          <button
            type="button"
            onClick={random}
            className="electric-button px-4 py-2 rounded-lg text-sm"
          >
            Pick random 5-digit
          </button>
        </div>
        {createIndex > 50 && (
          <p className="mt-2 text-amber-400/90 text-xs">
            Higher indices are fine — but remember to fund them.
          </p>
        )}
        <p className="mt-4 text-xs text-[#6b8fcc]">
          Derivation follows standard paths per chain. The agent will use this new identity going forward.
        </p>
        <button
          type="button"
          onClick={onPreview}
          className="mt-6 electric-button flex items-center gap-2 px-5 py-2.5 rounded-lg"
        >
          Preview Addresses
          <ChevronRight className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
}

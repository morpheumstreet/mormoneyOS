import { Loader2, Wand2 } from "lucide-react";
import { inputConfig } from "@/lib/theme";

interface SoulEnhancerSectionProps {
  enhanceWords: string;
  onEnhanceWordsChange: (value: string) => void;
  onPreview: () => void;
  onEnhanceAndApply: () => void;
  enhancing: boolean;
  hasWriteAccess: boolean;
}

export function SoulEnhancerSection({
  enhanceWords,
  onEnhanceWordsChange,
  onPreview,
  onEnhanceAndApply,
  enhancing,
  hasWriteAccess,
}: SoulEnhancerSectionProps) {
  const buttonClass =
    "electric-button flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium disabled:opacity-50";

  return (
    <div className="rounded-lg border border-[#29509c] bg-[#071228]/50 p-4 space-y-3">
      <label className="block text-sm font-medium text-[#8aa8df]">
        Soul enhancer
      </label>
      <p className="text-xs text-[#6b8fcc]">
        Enter at least 5 casual words. The AI will turn them into a complete,
        ready-to-use system prompt.
      </p>
      <div className="flex gap-2">
        <input
          type="text"
          value={enhanceWords}
          onChange={(e) => onEnhanceWordsChange(e.target.value)}
          disabled={!hasWriteAccess}
          placeholder="e.g. helpful financial assistant with warm tone and analytical style"
          className={`flex-1 ${inputConfig}`}
        />
        <button
          type="button"
          onClick={onPreview}
          disabled={enhancing || !hasWriteAccess}
          className={buttonClass}
        >
          {enhancing ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Wand2 className="h-4 w-4" />
          )}
          Preview
        </button>
        <button
          type="button"
          onClick={onEnhanceAndApply}
          disabled={enhancing || !hasWriteAccess}
          className={buttonClass}
        >
          {enhancing ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Wand2 className="h-4 w-4" />
          )}
          Enhance & Apply
        </button>
      </div>
    </div>
  );
}

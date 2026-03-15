import { useEffect } from "react";

interface CopyToastProps {
  onDone: () => void;
  duration?: number;
}

export function CopyToast({ onDone, duration = 2000 }: CopyToastProps) {
  useEffect(() => {
    const t = setTimeout(onDone, duration);
    return () => clearTimeout(t);
  }, [onDone, duration]);

  return (
    <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 electric-card px-4 py-2 border-[#00d4aa]/40 bg-[#00d4aa]/10 text-[#00d4aa] text-sm font-medium animate-[riseIn_0.3s_ease]">
      Copied!
    </div>
  );
}

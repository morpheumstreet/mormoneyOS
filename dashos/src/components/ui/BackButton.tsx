import { ArrowLeft } from "lucide-react";

interface BackButtonProps {
  onClick: () => void;
  children?: React.ReactNode;
}

export function BackButton({ onClick, children = "Back" }: BackButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex items-center gap-1 text-[#8aa8df] hover:text-white"
    >
      <ArrowLeft className="h-4 w-4" />
      {children}
    </button>
  );
}

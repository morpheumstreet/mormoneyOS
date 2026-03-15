import { filterButtonClass, filterButtonClassLg } from "@/lib/theme";

interface FilterButtonProps {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
  size?: "sm" | "lg";
}

export function FilterButton({
  active,
  onClick,
  children,
  size = "sm",
}: FilterButtonProps) {
  const cls =
    size === "lg"
      ? filterButtonClassLg(active)
      : filterButtonClass(active);
  return (
    <button type="button" onClick={onClick} className={cls}>
      {children}
    </button>
  );
}

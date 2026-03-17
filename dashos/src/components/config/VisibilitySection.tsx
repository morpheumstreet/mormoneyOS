import type { LucideIcon } from "lucide-react";
import { ToggleSwitch } from "@/components/ui/ToggleSwitch";

export interface VisibilityItem {
  id: string;
  icon: LucideIcon;
  label: string;
}

interface VisibilitySectionProps {
  title: string;
  description: string;
  items: VisibilityItem[];
  hiddenIds: Set<string>;
  onToggle: (id: string) => void;
  ariaLabelPrefix?: string;
}

export function VisibilitySection({
  title,
  description,
  items,
  hiddenIds,
  onToggle,
  ariaLabelPrefix = "menu",
}: VisibilitySectionProps) {
  return (
    <div className="electric-card overflow-hidden">
      <div className="border-b border-[#1a3670] px-4 py-3">
        <h3 className="text-sm font-semibold text-white">{title}</h3>
        <p className="mt-0.5 text-xs text-[#8aa8df]">{description}</p>
      </div>
      <div className="divide-y divide-[#1a3670]">
        {items.map((item) => {
          const Icon = item.icon;
          const visible = !hiddenIds.has(item.id);
          return (
            <div
              key={item.id}
              className="flex items-center justify-between gap-4 px-4 py-3 hover:bg-[#07132f]/50 transition-colors"
            >
              <div className="flex items-center gap-3 min-w-0 flex-1">
                <Icon className="h-4 w-4 shrink-0 text-[#9bb7eb]" />
                <span className="font-medium text-white">{item.label}</span>
              </div>
              <ToggleSwitch
                checked={visible}
                label={`${visible ? "Hide" : "Show"} ${item.label} in ${ariaLabelPrefix}`}
                onChange={() => onToggle(item.id)}
              />
            </div>
          );
        })}
      </div>
    </div>
  );
}

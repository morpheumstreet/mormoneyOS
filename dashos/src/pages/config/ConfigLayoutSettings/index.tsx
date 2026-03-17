import { LayoutGrid } from "lucide-react";
import { ConfigPageLayout } from "@/components/config/ConfigPageLayout";
import { VisibilitySection } from "@/components/config/VisibilitySection";
import { useVisibilityPreference } from "@/hooks/useVisibilityPreference";
import { configNavStorage, CONFIG_TOGGLEABLE_ITEMS } from "@/lib/configNav";
import {
  sidebarNavStorage,
  SIDEBAR_TOGGLEABLE_ITEMS,
} from "@/lib/sidebarNav";

const SECTIONS = [
  {
    title: "Config submenu",
    description: "Pages shown under Config",
    items: CONFIG_TOGGLEABLE_ITEMS,
    storage: configNavStorage,
    ariaLabelPrefix: "config menu",
  },
  {
    title: "Side menu",
    description: "Dashboard and Config are always shown. Others are customizable.",
    items: SIDEBAR_TOGGLEABLE_ITEMS,
    storage: sidebarNavStorage,
    ariaLabelPrefix: "side menu",
  },
] as const;

export default function ConfigLayoutSettings() {
  return (
    <ConfigPageLayout
      icon={LayoutGrid}
      title="Layout"
      description="Choose which pages appear in the Config submenu and side menu. Changes apply immediately."
      hasWriteAccess={true}
      error={null}
      loading={false}
    >
      <div className="space-y-6">
        {SECTIONS.map((section) => (
          <SectionWithVisibility
            key={section.title}
            {...section}
          />
        ))}
      </div>
    </ConfigPageLayout>
  );
}

function SectionWithVisibility({
  title,
  description,
  items,
  storage,
  ariaLabelPrefix,
}: (typeof SECTIONS)[number]) {
  const [hiddenIds, toggle] = useVisibilityPreference(storage);
  return (
    <VisibilitySection
      title={title}
      description={description}
      items={items}
      hiddenIds={hiddenIds}
      onToggle={toggle}
      ariaLabelPrefix={ariaLabelPrefix}
    />
  );
}

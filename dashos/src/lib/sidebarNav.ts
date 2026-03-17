import {
  FileText,
  LayoutDashboard,
  Puzzle,
  Settings,
  type LucideIcon,
} from "lucide-react";
import { createVisibilityStorage } from "./storageVisibility";

export interface SidebarNavItem {
  id: string;
  to: string;
  icon: LucideIcon;
  label: string;
  alwaysVisible?: boolean;
}

export const SIDEBAR_NAV_ITEMS: SidebarNavItem[] = [
  { id: "dashboard", to: "/", icon: LayoutDashboard, label: "Dashboard", alwaysVisible: true },
  { id: "reports", to: "/reports", icon: FileText, label: "Reports" },
  { id: "skills", to: "/skills", icon: Puzzle, label: "Skills" },
  { id: "config", to: "/config", icon: Settings, label: "Config", alwaysVisible: true },
];

export const SIDEBAR_NAV_CHANGE_EVENT = "dashos:sidebar-nav-changed";

export const sidebarNavStorage = createVisibilityStorage({
  storageKey: "dashos:sidebar-nav-hidden",
  changeEvent: SIDEBAR_NAV_CHANGE_EVENT,
});

export const getHiddenSidebarIds = sidebarNavStorage.getHiddenIds;
export const setHiddenSidebarIds = sidebarNavStorage.setHiddenIds;

export function getVisibleSidebarItems(): SidebarNavItem[] {
  const hidden = getHiddenSidebarIds();
  return SIDEBAR_NAV_ITEMS.filter(
    (item) => item.alwaysVisible || !hidden.has(item.id)
  );
}

export const SIDEBAR_TOGGLEABLE_ITEMS = SIDEBAR_NAV_ITEMS.filter(
  (item) => !item.alwaysVisible
);

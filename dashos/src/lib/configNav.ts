import {
  FileJson,
  Fish,
  Shield,
  Wrench,
  Users,
  Network,
  Cpu,
  Sparkles,
  Wallet,
  Wallet2,
  Activity,
  LayoutGrid,
  type LucideIcon,
} from "lucide-react";
import { createVisibilityStorage } from "./storageVisibility";

export const CONFIG_NAV_LAYOUT_ID = "layout";

export interface ConfigNavItem {
  id: string;
  to: string;
  icon: LucideIcon;
  label: string;
  alwaysVisible?: boolean;
  /** Hidden for guest users (read-only); exposes sensitive config */
  guestHidden?: boolean;
}

export const CONFIG_NAV_ITEMS: ConfigNavItem[] = [
  { id: CONFIG_NAV_LAYOUT_ID, to: "/config/layout", icon: LayoutGrid, label: "Layout", alwaysVisible: true },
  { id: "general", to: "/config/general", icon: FileJson, label: "General", guestHidden: true },
  { id: "auth", to: "/config/auth", icon: Shield, label: "Auth", guestHidden: true },
  { id: "tools", to: "/config/tools", icon: Wrench, label: "Tools" },
  { id: "social", to: "/config/social", icon: Users, label: "Social" },
  { id: "heartbeat", to: "/config/heartbeat", icon: Activity, label: "Heartbeat" },
  { id: "tunnel", to: "/config/tunnel", icon: Network, label: "Tunnel" },
  { id: "mirofish", to: "/config/mirofish", icon: Fish, label: "MiroFish" },
  { id: "model-list", to: "/config/model-list", icon: Cpu, label: "Model List", guestHidden: true },
  { id: "economic", to: "/config/economic", icon: Wallet, label: "Economic", guestHidden: true },
  { id: "wallet", to: "/config/wallet", icon: Wallet2, label: "ID", guestHidden: true },
  { id: "soul", to: "/config/soul", icon: Sparkles, label: "Soul", guestHidden: true },
];

export const CONFIG_NAV_CHANGE_EVENT = "dashos:config-nav-changed";

export const configNavStorage = createVisibilityStorage({
  storageKey: "dashos:config-nav-hidden",
  changeEvent: CONFIG_NAV_CHANGE_EVENT,
});

export const getHiddenNavIds = configNavStorage.getHiddenIds;
export const setHiddenNavIds = configNavStorage.setHiddenIds;

export function getVisibleNavItems(): ConfigNavItem[] {
  const hidden = getHiddenNavIds();
  return CONFIG_NAV_ITEMS.filter(
    (item) => item.alwaysVisible || !hidden.has(item.id)
  );
}

export function getVisibleNavItemsForGuest(): ConfigNavItem[] {
  return getVisibleNavItems().filter((item) => !item.guestHidden);
}

const GUEST_HIDDEN_PATHS = new Set(
  CONFIG_NAV_ITEMS.filter((i) => i.guestHidden).map((i) => i.to)
);

export function isGuestHiddenPath(path: string): boolean {
  return GUEST_HIDDEN_PATHS.has(path) || Array.from(GUEST_HIDDEN_PATHS).some((p) => path.startsWith(p + "/"));
}

export const CONFIG_TOGGLEABLE_ITEMS = CONFIG_NAV_ITEMS.filter(
  (item) => !item.alwaysVisible && item.id !== CONFIG_NAV_LAYOUT_ID
);

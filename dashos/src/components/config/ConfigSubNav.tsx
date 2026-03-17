import { NavLink } from "react-router-dom";
import { useStorageSync } from "@/hooks/useStorageSync";
import { getVisibleNavItems, CONFIG_NAV_CHANGE_EVENT } from "@/lib/configNav";

export default function ConfigSubNav() {
  const items = useStorageSync(getVisibleNavItems, CONFIG_NAV_CHANGE_EVENT);

  return (
    <nav className="config-sub-nav relative flex items-center gap-1 overflow-x-auto border-b border-[#1a3670] bg-[#050d1f]/95 px-4 py-2 backdrop-blur-sm md:px-6">
      <div className="absolute inset-0 pointer-events-none opacity-60 bg-[radial-gradient(circle_at_0%_50%,rgba(41,148,255,0.12),transparent_50%)]" />
      <div className="relative flex min-w-0 items-center gap-1">
        {items.map(({ to, icon: Icon, label }) => (
          <NavLink
            key={to}
            to={to}
            end={false}
            className={({ isActive }) =>
              [
                "group flex items-center gap-2 whitespace-nowrap rounded-lg border px-3 py-2 text-sm font-medium transition-all duration-200",
                isActive
                  ? "border-[#3a6de0] bg-[#0b2f80]/55 text-white shadow-[0_0_20px_-10px_rgba(72,140,255,0.8)]"
                  : "border-transparent text-[#9bb7eb] hover:border-[#294a8d] hover:bg-[#07132f]/80 hover:text-white",
              ].join(" ")
            }
          >
            <Icon className="h-4 w-4 shrink-0 opacity-90" />
            <span>{label}</span>
          </NavLink>
        ))}
      </div>
    </nav>
  );
}

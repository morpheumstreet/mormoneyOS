import { NavLink } from "react-router-dom";
import {
  FileJson,
  Wrench,
  Users,
  Network,
  Cpu,
  Sparkles,
  Wallet,
  Wallet2,
  Activity,
} from "lucide-react";

const configNavItems = [
  { to: "/config/general", icon: FileJson, label: "General" },
  { to: "/config/tools", icon: Wrench, label: "Tools" },
  { to: "/config/social", icon: Users, label: "Social" },
  { to: "/config/heartbeat", icon: Activity, label: "Heartbeat" },
  { to: "/config/tunnel", icon: Network, label: "Tunnel" },
  { to: "/config/model-list", icon: Cpu, label: "Model List" },
  { to: "/config/economic", icon: Wallet, label: "Economic" },
  { to: "/config/wallet", icon: Wallet2, label: "Wallet" },
  { to: "/config/soul", icon: Sparkles, label: "Soul" },
];

export default function ConfigSubNav() {
  return (
    <nav className="config-sub-nav relative flex items-center gap-1 overflow-x-auto border-b border-[#1a3670] bg-[#050d1f]/95 px-4 py-2 backdrop-blur-sm md:px-6">
      <div className="absolute inset-0 pointer-events-none opacity-60 bg-[radial-gradient(circle_at_0%_50%,rgba(41,148,255,0.12),transparent_50%)]" />
      <div className="relative flex min-w-0 items-center gap-1">
        {configNavItems.map(({ to, icon: Icon, label }) => (
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

import { useLocation } from 'react-router-dom';
import { LogOut, Menu, PanelLeftClose, PanelLeftOpen, Wallet } from 'lucide-react';
import { useWalletAuth } from '@/contexts/WalletAuthContext';

const routeTitles: Record<string, string> = {
  '/': 'Dashboard',
  '/agent': 'Agent Chat',
  '/config': 'Config',
};

interface HeaderProps {
  isSidebarCollapsed: boolean;
  onToggleSidebar: () => void;
  onToggleSidebarCollapse: () => void;
}

export default function Header({
  isSidebarCollapsed,
  onToggleSidebar,
  onToggleSidebarCollapse,
}: HeaderProps) {
  const location = useLocation();
  const { address, isGuest, isAuthenticated, disconnect } = useWalletAuth();
  const pageTitle =
    routeTitles[location.pathname] ??
    (location.pathname.startsWith('/config') ? 'Config' : 'Dashboard');

  const shortAddress = isGuest
    ? 'Guest'
    : address
      ? address.length > 12
        ? `${address.slice(0, 6)}…${address.slice(-4)}`
        : address
      : '—';

  return (
    <header className="glass-header relative flex min-h-[4.5rem] flex-wrap items-center justify-between gap-2 border border-[#1a3670] px-4 py-3 sm:px-5 sm:py-3.5 md:flex-nowrap md:px-8 md:py-4">
      <div className="absolute inset-0 pointer-events-none opacity-70 bg-[radial-gradient(circle_at_15%_30%,rgba(41,148,255,0.22),transparent_45%),radial-gradient(circle_at_85%_75%,rgba(0,209,255,0.14),transparent_40%)]" />

      <div className="relative flex min-w-0 items-center gap-2.5 sm:gap-3">
        <button
          type="button"
          onClick={onToggleSidebar}
          aria-label="Open navigation"
          className="rounded-lg border border-[#294a8f] bg-[#081637]/70 p-1.5 text-[#9ec2ff] transition hover:border-[#4f83ff] hover:text-white md:hidden"
        >
          <Menu className="h-5 w-5" />
        </button>

        <div className="min-w-0">
          <h1 className="truncate text-base font-semibold tracking-wide text-white sm:text-lg">
            {pageTitle}
          </h1>
          <p className="hidden text-[10px] uppercase tracking-[0.16em] text-[#7ea5eb] sm:block">
            MoneyClaw Command Center
          </p>
        </div>
      </div>

      <div className="relative hidden w-full items-center justify-end gap-1.5 sm:gap-2 md:flex md:w-auto md:gap-3">
        <button
          type="button"
          onClick={onToggleSidebarCollapse}
          className="hidden items-center gap-1 rounded-lg border border-[#2b4f97] bg-[#091937]/75 px-2.5 py-1.5 text-xs text-[#c4d8ff] transition hover:border-[#4f83ff] hover:text-white md:flex md:text-sm"
          title={isSidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        >
          {isSidebarCollapsed ? <PanelLeftOpen className="h-4 w-4" /> : <PanelLeftClose className="h-4 w-4" />}
          <span>{isSidebarCollapsed ? 'Expand' : 'Collapse'}</span>
        </button>

        {isAuthenticated && (
          <div className="hidden items-center gap-2 rounded-lg border border-[#2b4f97] bg-[#091937]/75 px-2.5 py-1.5 text-xs text-[#c4d8ff] md:flex">
            <Wallet className="h-3.5 w-3.5" />
            <span className={isGuest ? '' : 'font-mono'}>{shortAddress}{isGuest ? ' (read-only)' : ''}</span>
          </div>
        )}

        <button
          type="button"
          onClick={disconnect}
          className="hidden items-center gap-1 rounded-lg border border-[#2b4f97] bg-[#091937]/75 px-2.5 py-1.5 text-xs text-[#c4d8ff] transition hover:border-[#4f83ff] hover:text-white sm:gap-1.5 sm:px-3 sm:text-sm md:flex"
        >
          <LogOut className="h-4 w-4" />
          <span className="hidden sm:inline">Disconnect</span>
        </button>
      </div>
    </header>
  );
}

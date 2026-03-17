import { useEffect, useState } from 'react';
import { NavLink } from 'react-router-dom';
import { ChevronsLeftRightEllipsis, LogOut, Wallet, X } from 'lucide-react';
import { useStorageSync } from '@/hooks/useStorageSync';
import { getVisibleSidebarItems, SIDEBAR_NAV_CHANGE_EVENT } from '@/lib/sidebarNav';
import { getVersion, type VersionResponse } from '@/lib/api';
import { useWalletAuth } from '@/contexts/WalletAuthContext';

const COLLAPSE_BUTTON_DELAY_MS = 1000;

interface SidebarProps {
  isOpen: boolean;
  isCollapsed: boolean;
  onClose: () => void;
  onToggleCollapse: () => void;
}

export default function Sidebar({ isOpen, isCollapsed, onClose, onToggleCollapse }: SidebarProps) {
  const [showCollapseButton, setShowCollapseButton] = useState(false);
  const [versionInfo, setVersionInfo] = useState<VersionResponse | null>(null);
  const navItems = useStorageSync(getVisibleSidebarItems, SIDEBAR_NAV_CHANGE_EVENT);
  const { address, isGuest, isAuthenticated, disconnect } = useWalletAuth();
  const shortAddress = isGuest
    ? 'Guest'
    : address
      ? address.length > 12
        ? `${address.slice(0, 6)}…${address.slice(-4)}`
        : address
      : '—';

  useEffect(() => {
    const id = setTimeout(() => setShowCollapseButton(true), COLLAPSE_BUTTON_DELAY_MS);
    return () => clearTimeout(id);
  }, []);

  useEffect(() => {
    getVersion()
      .then(setVersionInfo)
      .catch(() => setVersionInfo(null));
  }, []);

  return (
    <>
      <button
        type="button"
        aria-label="Close navigation"
        onClick={onClose}
        className={[
          'fixed inset-0 z-30 bg-black/50 transition-opacity md:hidden',
          isOpen ? 'opacity-100' : 'pointer-events-none opacity-0',
        ].join(' ')}
      />
      <aside
        className={[
          'fixed left-0 top-0 z-40 flex h-screen w-[86vw] max-w-[17.5rem] flex-col border-r border-[#1e2f5d] bg-[#050b1a]/95 backdrop-blur-xl',
          'shadow-[0_0_50px_-25px_rgba(8,121,255,0.7)]',
          'transform transition-[width,transform] duration-300 ease-out',
          isOpen ? 'translate-x-0' : '-translate-x-full',
          isCollapsed ? 'md:w-[6.25rem]' : 'md:w-[17.5rem]',
          'md:translate-x-0',
        ].join(' ')}
      >
        <div className="relative flex items-center justify-between border-b border-[#1a2d5e] px-4 py-4">
          <div className="flex items-center gap-3 overflow-hidden">
            {!isCollapsed && (
              <>
                <div
                  className="electric-brand-mark h-9 w-9 shrink-0 rounded-xl"
                  role="img"
                  aria-label="MormOS"
                >
                  <span className="sr-only">MormOS</span>
                </div>
                <span className="text-lg font-semibold tracking-[0.1em] text-white">MormOS</span>
              </>
            )}
          </div>

          <div className="flex items-center gap-2">
            {showCollapseButton && (
              <button
                type="button"
                onClick={onToggleCollapse}
                aria-label={isCollapsed ? 'Expand navigation' : 'Collapse navigation'}
                className="hidden rounded-lg border border-[#2c4e97] bg-[#0a1b3f]/60 p-1.5 text-[#8bb9ff] transition hover:border-[#4f83ff] hover:text-white md:block"
              >
                <ChevronsLeftRightEllipsis className="h-4 w-4" />
              </button>
            )}
            <button
              type="button"
              onClick={onClose}
              aria-label="Close navigation"
              className="rounded-lg p-1.5 text-gray-300 transition-colors hover:bg-gray-800 hover:text-white md:hidden"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        </div>

        <nav className="flex-1 space-y-1 overflow-y-auto px-3 py-4">
          {navItems.map(({ to, icon: Icon, label }) => (
            <NavLink
              key={to}
              to={to}
              end={to === '/'}
              onClick={onClose}
              title={isCollapsed ? label : undefined}
              className={({ isActive }) =>
                [
                  'group flex items-center gap-3 overflow-hidden rounded-xl px-3 py-2.5 text-sm font-medium transition-all duration-300',
                  isActive
                    ? 'border border-[#3a6de0] bg-[#0b2f80]/55 text-white shadow-[0_0_30px_-16px_rgba(72,140,255,0.95)]'
                    : 'border border-transparent text-[#9bb7eb] hover:border-[#294a8d] hover:bg-[#07132f] hover:text-white',
                ].join(' ')
              }
            >
              <Icon className="h-5 w-5 shrink-0 transition-transform duration-300 group-hover:scale-110" />
              <span
                className={[
                  'whitespace-nowrap transition-[opacity,transform,width] duration-300',
                  isCollapsed ? 'w-0 -translate-x-3 opacity-0 md:invisible' : 'w-auto opacity-100',
                ].join(' ')}
              >
                {label}
              </span>
            </NavLink>
          ))}
        </nav>

        <div className="mx-3 mb-3 flex flex-col gap-2 md:hidden">
          {isAuthenticated && (
            <div className="flex items-center gap-2 rounded-lg border border-[#2b4f97] bg-[#091937]/75 px-2.5 py-1.5 text-xs text-[#c4d8ff]">
              <Wallet className="h-3.5 w-3.5 shrink-0" />
              <span className={isGuest ? '' : 'font-mono truncate'}>{shortAddress}{isGuest ? ' (read-only)' : ''}</span>
            </div>
          )}
          <button
            type="button"
            onClick={disconnect}
            className="flex items-center justify-center gap-1.5 rounded-lg border border-[#2b4f97] bg-[#091937]/75 px-3 py-2 text-xs text-[#c4d8ff] transition hover:border-[#4f83ff] hover:text-white"
          >
            <LogOut className="h-4 w-4" />
            <span>Disconnect</span>
          </button>
        </div>

        <div
          className={[
            'mx-3 mb-4 rounded-xl border border-[#1b3670] bg-[#071328]/80 px-3 py-3 text-xs text-[#89a9df] transition-all duration-300',
            isCollapsed ? 'md:px-1.5 md:text-center' : '',
          ].join(' ')}
        >
          <p className={isCollapsed ? 'hidden md:block' : ''}>mormOS</p>
          <p className={isCollapsed ? 'text-[10px] uppercase tracking-widest' : 'mt-1 text-[#5f84cc]'}>
            {isCollapsed ? 'UI' : 'Command Center'}
          </p>
          {versionInfo && (
            <p
              className={[
                'mt-2 truncate font-mono text-[10px] text-[#4a6ba8]',
                isCollapsed ? 'hidden md:block md:truncate' : '',
              ].join(' ')}
              title={
                [versionInfo.version, versionInfo.commit, versionInfo.build_time]
                  .filter(Boolean)
                  .join(' · ') || undefined
              }
            >
              {versionInfo.version || 'dev'}
              {versionInfo.commit ? ` · ${versionInfo.commit}` : ''}
            </p>
          )}
        </div>
      </aside>
    </>
  );
}

import { Save } from "lucide-react";
import { Shield } from "lucide-react";
import { AlertTriangle } from "lucide-react";
import { useWalletAuth } from "@/contexts/WalletAuthContext";
import { ConfigPageLayout } from "@/components/config/ConfigPageLayout";
import { FormInput } from "@/components/ui/FormInput";
import { ToggleSwitch } from "@/components/ui/ToggleSwitch";
import { useAuthConfig } from "./useAuthConfig";

const GUEST_ACCESS_WARNING =
  "Guest access allows anyone to view the dashboard without signing in. " +
  "Guests cannot make any changes, write requests are blocked, and sensitive configuration (API keys, wallet details, etc.) is hidden. " +
  "Are you sure you want to enable guest access?";

export default function ConfigAuth() {
  const { hasWriteAccess } = useWalletAuth();
  const {
    config,
    loading,
    saving,
    error,
    success,
    updateConfig,
    handleSave,
  } = useAuthConfig(!!hasWriteAccess);

  const handleGuestToggle = () => {
    if (config.guest_access_enabled) {
      updateConfig("guest_access_enabled", false);
      return;
    }
    if (window.confirm(GUEST_ACCESS_WARNING)) {
      updateConfig("guest_access_enabled", true);
    }
  };

  const headerActions = (
    <button
      onClick={handleSave}
      disabled={saving || !hasWriteAccess}
      className="electric-button flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium disabled:opacity-50"
    >
      <Save className="h-4 w-4" />
      {saving ? "Saving…" : "Save"}
    </button>
  );

  return (
    <ConfigPageLayout
      icon={Shield}
      title="Auth"
      description="Dashboard sign-in — restrict write access to a specific wallet address"
      hasWriteAccess={!!hasWriteAccess}
      writeAccessMessage="Connect your wallet and sign to configure auth."
      error={error}
      loading={loading}
      success={success}
      headerActions={headerActions}
    >
      <div className="electric-card p-6 space-y-6">
        <FormInput
          label="Creator address"
          help="Ethereum address (0x...) that can sign in for write access. Leave empty to allow any wallet."
          value={config.creator_address}
          onChange={(e) => updateConfig("creator_address", e.target.value.trim())}
          disabled={!hasWriteAccess}
          placeholder="0x..."
        />
        <p className="text-xs text-[#6b8fcc]">
          When set, only this address can connect a wallet and sign the SIWE message to get write access.
          When empty, any wallet can sign in. The agent wallet (walletAddress) is separate from creator.
        </p>

        <div className="pt-4 border-t border-[#1a3670]">
          <div className="flex items-start justify-between gap-4">
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <label className="text-sm font-medium text-[#8aa8df]">
                  Guest access
                </label>
                <AlertTriangle className="h-4 w-4 text-amber-400 shrink-0" />
              </div>
              <p className="mt-1 text-xs text-[#6b8fcc]">
                Allow unauthenticated users to view the dashboard in read-only mode. No write requests, no sensitive config.
              </p>
            </div>
            <ToggleSwitch
              checked={!!config.guest_access_enabled}
              disabled={!hasWriteAccess}
              label="Enable guest access"
              onChange={handleGuestToggle}
            />
          </div>
        </div>
      </div>
    </ConfigPageLayout>
  );
}

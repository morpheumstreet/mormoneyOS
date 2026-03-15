import { Save } from "lucide-react";
import { Wallet } from "lucide-react";
import { useWalletAuth } from "@/contexts/WalletAuthContext";
import { ConfigPageLayout } from "@/components/config/ConfigPageLayout";
import { useEconomicConfig } from "./useEconomicConfig";
import { BalancesTable } from "./BalancesTable";
import { TreasuryPolicyForm } from "./TreasuryPolicyForm";

export default function ConfigEconomic() {
  const { hasWriteAccess } = useWalletAuth();
  const {
    data,
    loading,
    saving,
    error,
    success,
    resourceMode,
    setResourceMode,
    treasury,
    updateTreasuryField,
    handleSave,
  } = useEconomicConfig(!!hasWriteAccess);

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
      icon={Wallet}
      title="Economic"
      description="Wallet addresses, USDC balances, and economic constraints"
      hasWriteAccess={!!hasWriteAccess}
      writeAccessMessage="Connect your wallet and sign to edit economic constraints."
      error={error}
      loading={loading}
      success={success}
      headerActions={headerActions}
    >
      <BalancesTable balances={data?.balances ?? []} />
      <TreasuryPolicyForm
        resourceMode={resourceMode}
        treasury={treasury}
        disabled={!hasWriteAccess}
        onResourceModeChange={setResourceMode}
        onTreasuryChange={updateTreasuryField}
      />
    </ConfigPageLayout>
  );
}

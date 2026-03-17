import { Save } from "lucide-react";
import { Fish } from "lucide-react";
import { useWalletAuth } from "@/contexts/WalletAuthContext";
import { ConfigPageLayout } from "@/components/config/ConfigPageLayout";
import { FormInput } from "@/components/ui/FormInput";
import { FormCheckbox } from "@/components/ui/FormCheckbox";
import { useMiroFishConfig } from "./useMiroFishConfig";

export default function ConfigMiroFish() {
  const { hasWriteAccess } = useWalletAuth();
  const {
    config,
    loading,
    saving,
    error,
    success,
    updateConfig,
    handleSave,
  } = useMiroFishConfig(!!hasWriteAccess);

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
      icon={Fish}
      title="MiroFish"
      description="Swarm intelligence / foresight layer — run simulations, get reports, chat with digital crowd"
      hasWriteAccess={!!hasWriteAccess}
      writeAccessMessage="Connect your wallet and sign to configure MiroFish."
      error={error}
      loading={loading}
      success={success}
      headerActions={headerActions}
    >
      <div className="electric-card p-6 space-y-6">
        <div className="space-y-4">
          <FormCheckbox
            label="Enable MiroFish"
            checked={config.enabled}
            onChange={(e) => updateConfig("enabled", e.target.checked)}
            disabled={!hasWriteAccess}
          />
          <p className="text-xs text-[#6b8fcc]">
            When enabled, the agent can use the mirofish tool for market predictions, foresight, and rehearsal before betting.
          </p>
        </div>

        <FormInput
          label="Base URL"
          help="MiroFish API base URL (e.g. http://localhost:5001 or http://mirofish:5001 in Docker)"
          type="url"
          value={config.base_url}
          onChange={(e) => updateConfig("base_url", e.target.value)}
          disabled={!hasWriteAccess}
          placeholder="http://localhost:5001"
        />

        <FormInput
          label="Timeout (seconds)"
          help="Request timeout for MiroFish API calls"
          type="number"
          min={10}
          max={3600}
          value={config.timeout_seconds}
          onChange={(e) => updateConfig("timeout_seconds", parseInt(e.target.value, 10) || 300)}
          disabled={!hasWriteAccess}
        />

        <FormInput
          label="Default LLM"
          help="LLM model for swarm simulations (e.g. qwen-plus)"
          value={config.default_llm}
          onChange={(e) => updateConfig("default_llm", e.target.value)}
          disabled={!hasWriteAccess}
          placeholder="qwen-plus"
        />

        <FormInput
          label="Max agents"
          help="Maximum number of agents in swarm simulations"
          type="number"
          min={1}
          max={10000}
          value={config.max_agents}
          onChange={(e) => updateConfig("max_agents", parseInt(e.target.value, 10) || 2000)}
          disabled={!hasWriteAccess}
        />
      </div>
    </ConfigPageLayout>
  );
}

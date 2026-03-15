import { Save, Sparkles } from "lucide-react";
import { useWalletAuth } from "@/contexts/WalletAuthContext";
import { ConfigPageLayout } from "@/components/config/ConfigPageLayout";
import { useSoulConfig } from "./useSoulConfig";
import { SoulEnhancerSection } from "./SoulEnhancerSection";
import { SoulFormFields } from "./SoulFormFields";

export default function ConfigSoul() {
  const { hasWriteAccess } = useWalletAuth();
  const {
    config,
    loading,
    saving,
    error,
    success,
    enhanceWords,
    setEnhanceWords,
    enhancing,
    handleSave,
    handleEnhance,
    updateConfig,
    updateConstraint,
    addConstraint,
    removeConstraint,
  } = useSoulConfig(!!hasWriteAccess);

  return (
    <ConfigPageLayout
      icon={Sparkles}
      title="Soul"
      description="Agent personality, system prompt, tone, and behavioral constraints"
      hasWriteAccess={!!hasWriteAccess}
      writeAccessMessage="Connect your wallet and sign to edit soul configuration."
      error={error}
      success={success}
      loading={loading}
      headerActions={
        <button
          onClick={handleSave}
          disabled={saving || !hasWriteAccess}
          className="electric-button flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium disabled:opacity-50"
        >
          <Save className="h-4 w-4" />
          {saving ? "Saving…" : "Save"}
        </button>
      }
    >
      <div className="electric-card p-6 space-y-6">
        <SoulEnhancerSection
          enhanceWords={enhanceWords}
          onEnhanceWordsChange={setEnhanceWords}
          onPreview={() => handleEnhance(false)}
          onEnhanceAndApply={() => handleEnhance(true)}
          enhancing={enhancing}
          hasWriteAccess={!!hasWriteAccess}
        />

        <SoulFormFields
          config={config}
          hasWriteAccess={!!hasWriteAccess}
          onConfigChange={updateConfig}
          onConstraintUpdate={updateConstraint}
          onConstraintAdd={addConstraint}
          onConstraintRemove={removeConstraint}
        />
      </div>
    </ConfigPageLayout>
  );
}

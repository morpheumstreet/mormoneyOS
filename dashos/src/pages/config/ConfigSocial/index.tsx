import { Users } from "lucide-react";
import { useWalletAuth } from "@/contexts/WalletAuthContext";
import { ConfigPageLayout } from "@/components/config/ConfigPageLayout";
import { useSocialConfig } from "./useSocialConfig";
import { SocialChannelRow } from "./SocialChannelRow";

export default function ConfigSocial() {
  const { hasWriteAccess } = useWalletAuth();
  const {
    channels,
    loading,
    error,
    toggling,
    expanded,
    formValues,
    saving,
    handleToggle,
    handleSaveConfig,
    updateFormValue,
    toggleExpanded,
  } = useSocialConfig(!!hasWriteAccess);

  return (
    <ConfigPageLayout
      icon={Users}
      title="Social"
      description="Enable channels and configure API keys. Restart required to apply changes."
      hasWriteAccess={!!hasWriteAccess}
      writeAccessMessage="Connect your wallet and sign to configure channels."
      error={error}
      loading={loading}
    >
      <div className="electric-card overflow-hidden">
        <div className="divide-y divide-[#1a3670]">
          {channels.length === 0 ? (
            <div className="px-4 py-8 text-center text-sm text-[#8aa8df]">
              No social channels available.
            </div>
          ) : (
            channels.map((channel) => (
              <SocialChannelRow
                key={channel.name}
                channel={channel}
                expanded={!!expanded[channel.name]}
                formValues={formValues[channel.name] || {}}
                toggling={!!toggling[channel.name]}
                saving={!!saving[channel.name]}
                hasWriteAccess={!!hasWriteAccess}
                onToggleExpand={() => toggleExpanded(channel.name)}
                onToggleEnabled={() => handleToggle(channel)}
                onSaveConfig={() => handleSaveConfig(channel)}
                onFormValueChange={(key, value) =>
                  updateFormValue(channel.name, key, value)
                }
              />
            ))
          )}
        </div>
      </div>
    </ConfigPageLayout>
  );
}

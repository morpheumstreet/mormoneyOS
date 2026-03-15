import { Network } from "lucide-react";
import { useWalletAuth } from "@/contexts/WalletAuthContext";
import { ConfigPageLayout } from "@/components/config/ConfigPageLayout";
import { useTunnelProviders } from "./useTunnelProviders";
import { ActiveTunnelsList } from "./ActiveTunnelsList";
import { TunnelProviderRow } from "./TunnelProviderRow";

export default function ConfigTunnel() {
  const { hasWriteAccess } = useWalletAuth();
  const state = useTunnelProviders(!!hasWriteAccess);

  return (
    <ConfigPageLayout
      icon={Network}
      title="Tunnel"
      description="Configure tunnel providers (bore, cloudflare, ngrok, tailscale). API keys required for cloudflare, ngrok, tailscale."
      hasWriteAccess={!!hasWriteAccess}
      writeAccessMessage="Connect your wallet and sign to configure tunnel providers."
      error={state.error}
      loading={state.loading}
    >
      <ActiveTunnelsList tunnels={state.tunnels} />

      <div className="electric-card overflow-hidden">
        <div className="divide-y divide-[#1a3670]">
          {!state.providersData?.providers?.length ? (
            <div className="px-4 py-8 text-center text-sm text-[#8aa8df]">
              No tunnel providers available.
            </div>
          ) : (
            state.providersData.providers.map((name) => {
              const schema = state.providersData!.schemas[name];
              const pc = state.providersData!.config?.providers?.[name] as Record<string, unknown> | undefined;
              const hasAuthFields = schema?.fields?.some((f) => f.type === "password") ?? false;
              const needsConfig =
                hasAuthFields &&
                !schema?.fields?.every((f) => {
                  if (f.type !== "password") return true;
                  const v = pc?.[f.name];
                  return typeof v === "string" && v !== "" && v !== "***";
                });

              return (
                <TunnelProviderRow
                  key={name}
                  name={name}
                  schema={schema}
                  providerConfig={pc}
                  expanded={!!state.expanded[name]}
                  formValues={state.formValues[name] || {}}
                  saving={!!state.saving[name]}
                  restarting={!!state.restarting[name]}
                  toggling={!!state.toggling[name]}
                  hasWriteAccess={!!hasWriteAccess}
                  needsConfig={!!needsConfig}
                  isConfigured={state.isConfigured}
                  onToggleExpand={() => state.setExpandedByName(name)}
                  onToggleEnabled={() => state.handleToggleEnabled(name)}
                  onRestart={() => state.handleRestart(name)}
                  onSaveConfig={() => state.handleSaveConfig(name)}
                  onFormValueChange={(key, value) =>
                    state.updateFormValue(name, key, value)
                  }
                />
              );
            })
          )}
        </div>
      </div>
    </ConfigPageLayout>
  );
}

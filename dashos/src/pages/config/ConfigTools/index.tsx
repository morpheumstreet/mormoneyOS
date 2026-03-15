import { Wrench } from "lucide-react";
import { useWalletAuth } from "@/contexts/WalletAuthContext";
import { ConfigPageLayout } from "@/components/config/ConfigPageLayout";
import { ToggleSwitch } from "@/components/ui/ToggleSwitch";
import { useToolsConfig } from "./useToolsConfig";

const EMPTY_MESSAGE =
  "No tools available. Ensure the agent is running with tools configured.";

export default function ConfigTools() {
  const { hasWriteAccess } = useWalletAuth();
  const { tools, loading, error, toggling, handleToggle } = useToolsConfig(
    !!hasWriteAccess
  );

  return (
    <ConfigPageLayout
      icon={Wrench}
      title="Tools"
      description="Enable or disable agent tools. Disabled tools are hidden from the LLM."
      hasWriteAccess={!!hasWriteAccess}
      writeAccessMessage="Connect your wallet and sign to toggle tools."
      error={error}
      loading={loading}
    >
      <div className="electric-card overflow-hidden">
        <div className="divide-y divide-[#1a3670]">
          {tools.length === 0 ? (
            <div className="px-4 py-8 text-center text-sm text-[#8aa8df]">
              {EMPTY_MESSAGE}
            </div>
          ) : (
            tools.map((tool) => (
              <div
                key={tool.name}
                className="flex items-center justify-between gap-4 px-4 py-3 hover:bg-[#07132f]/50 transition-colors"
              >
                <div className="min-w-0 flex-1">
                  <p className="font-medium text-white">{tool.name}</p>
                  {tool.description && (
                    <p className="mt-0.5 text-xs text-[#8aa8df] line-clamp-2">
                      {tool.description}
                    </p>
                  )}
                </div>
                <ToggleSwitch
                  checked={tool.enabled}
                  disabled={!hasWriteAccess}
                  loading={!!toggling[tool.name]}
                  label={`${tool.enabled ? "Disable" : "Enable"} ${tool.name}`}
                  onChange={() => handleToggle(tool)}
                />
              </div>
            ))
          )}
        </div>
      </div>
    </ConfigPageLayout>
  );
}

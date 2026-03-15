import { Activity } from "lucide-react";
import { useWalletAuth } from "@/contexts/WalletAuthContext";
import { ConfigPageLayout } from "@/components/config/ConfigPageLayout";
import { useHeartbeatConfig } from "./useHeartbeatConfig";
import { ScheduleItemRow } from "./ScheduleItemRow";

const EMPTY_MESSAGE =
  "No heartbeat schedules. Ensure the agent is running with a heartbeat store.";

export default function ConfigHeartbeat() {
  const { hasWriteAccess } = useWalletAuth();
  const {
    schedules,
    loading,
    error,
    toggling,
    expanded,
    scheduleEdits,
    savingSchedule,
    handleToggle,
    handleSaveSchedule,
    updateScheduleEdit,
    toggleExpanded,
  } = useHeartbeatConfig(!!hasWriteAccess);

  return (
    <ConfigPageLayout
      icon={Activity}
      title="Heartbeat"
      description="Cron jobs that run in the background. Enable/disable or change schedules."
      hasWriteAccess={!!hasWriteAccess}
      writeAccessMessage="Connect your wallet and sign to modify heartbeat schedules."
      error={error}
      loading={loading}
    >
      <div className="electric-card overflow-hidden">
        <div className="divide-y divide-[#1a3670]">
          {schedules.length === 0 ? (
            <div className="px-4 py-8 text-center text-sm text-[#8aa8df]">
              {EMPTY_MESSAGE}
            </div>
          ) : (
            schedules.map((item) => (
              <ScheduleItemRow
                key={item.name}
                item={item}
                expanded={!!expanded[item.name]}
                scheduleEdit={scheduleEdits[item.name] ?? item.schedule}
                toggling={!!toggling[item.name]}
                saving={!!savingSchedule[item.name]}
                hasWriteAccess={!!hasWriteAccess}
                onToggleExpand={() => toggleExpanded(item.name)}
                onToggleEnabled={() => handleToggle(item)}
                onScheduleChange={(value) => updateScheduleEdit(item.name, value)}
                onSaveSchedule={() => handleSaveSchedule(item)}
              />
            ))
          )}
        </div>
      </div>
    </ConfigPageLayout>
  );
}

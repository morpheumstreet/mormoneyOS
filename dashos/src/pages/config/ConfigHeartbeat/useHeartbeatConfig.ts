import { useEffect, useState, useCallback } from "react";
import {
  getHeartbeat,
  patchHeartbeatEnabled,
  patchHeartbeatSchedule,
  type HeartbeatScheduleItem,
} from "@/lib/api";
import { getApiErrorMessage } from "@/lib/api-error";
import { isValidCron } from "./cronUtils";

export function useHeartbeatConfig(hasWriteAccess: boolean) {
  const [schedules, setSchedules] = useState<HeartbeatScheduleItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [toggling, setToggling] = useState<Record<string, boolean>>({});
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});
  const [scheduleEdits, setScheduleEdits] = useState<Record<string, string>>({});
  const [savingSchedule, setSavingSchedule] = useState<Record<string, boolean>>({});

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    getHeartbeat()
      .then((res) => {
        setSchedules(res.schedules || []);
        const edits: Record<string, string> = {};
        for (const s of res.schedules || []) {
          edits[s.name] = s.schedule;
        }
        setScheduleEdits(edits);
      })
      .catch((e) => setError(getApiErrorMessage(e, "Load failed")))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const handleToggle = useCallback(
    async (item: HeartbeatScheduleItem) => {
      if (!hasWriteAccess) return;
      const nextEnabled = !item.enabled;
      setToggling((prev) => ({ ...prev, [item.name]: true }));
      try {
        await patchHeartbeatEnabled(item.name, nextEnabled);
        setSchedules((prev) =>
          prev.map((s) =>
            s.name === item.name ? { ...s, enabled: nextEnabled } : s
          )
        );
      } catch (e) {
        setError(getApiErrorMessage(e, "Update failed"));
      } finally {
        setToggling((prev) => ({ ...prev, [item.name]: false }));
      }
    },
    [hasWriteAccess]
  );

  const handleSaveSchedule = useCallback(
    async (item: HeartbeatScheduleItem) => {
      if (!hasWriteAccess) return;
      const schedule = scheduleEdits[item.name]?.trim();
      if (!schedule) {
        setError("Schedule cannot be empty");
        return;
      }
      if (!isValidCron(schedule)) {
        setError("Invalid cron expression");
        return;
      }
      setSavingSchedule((prev) => ({ ...prev, [item.name]: true }));
      setError(null);
      try {
        await patchHeartbeatSchedule(item.name, schedule);
        setSchedules((prev) =>
          prev.map((s) =>
            s.name === item.name ? { ...s, schedule } : s
          )
        );
      } catch (e) {
        setError(getApiErrorMessage(e, "Save failed"));
      } finally {
        setSavingSchedule((prev) => ({ ...prev, [item.name]: false }));
      }
    },
    [hasWriteAccess, scheduleEdits]
  );

  const updateScheduleEdit = useCallback((name: string, value: string) => {
    setScheduleEdits((prev) => ({ ...prev, [name]: value }));
  }, []);

  const toggleExpanded = useCallback((name: string) => {
    setExpanded((prev) =>
      prev[name] ? {} : { [name]: true }
    );
  }, []);

  return {
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
  };
}

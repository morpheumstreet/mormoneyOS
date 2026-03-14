import { useEffect, useState } from "react";
import {
  Activity,
  AlertTriangle,
  Loader2,
  ChevronDown,
  ChevronUp,
  Save,
} from "lucide-react";
import { useWalletAuth } from "@/contexts/WalletAuthContext";
import {
  getHeartbeat,
  patchHeartbeatEnabled,
  patchHeartbeatSchedule,
  type HeartbeatScheduleItem,
} from "@/lib/api";

function formatDateTime(s: string): string {
  if (!s) return "—";
  try {
    const d = new Date(s);
    return isNaN(d.getTime()) ? s : d.toLocaleString();
  } catch {
    return s;
  }
}

export default function ConfigHeartbeat() {
  const { hasWriteAccess } = useWalletAuth();
  const [schedules, setSchedules] = useState<HeartbeatScheduleItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [toggling, setToggling] = useState<Record<string, boolean>>({});
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});
  const [scheduleEdits, setScheduleEdits] = useState<Record<string, string>>({});
  const [savingSchedule, setSavingSchedule] = useState<Record<string, boolean>>({});

  const load = () => {
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
      .catch((e) => setError(e instanceof Error ? e.message : "Load failed"))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
  }, []);

  const handleToggle = async (item: HeartbeatScheduleItem) => {
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
      setError(e instanceof Error ? e.message : "Update failed");
    } finally {
      setToggling((prev) => ({ ...prev, [item.name]: false }));
    }
  };

  const handleSaveSchedule = async (item: HeartbeatScheduleItem) => {
    if (!hasWriteAccess) return;
    const schedule = scheduleEdits[item.name]?.trim();
    if (!schedule) {
      setError("Schedule cannot be empty");
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
      setError(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSavingSchedule((prev) => ({ ...prev, [item.name]: false }));
    }
  };

  const updateScheduleEdit = (name: string, value: string) => {
    setScheduleEdits((prev) => ({ ...prev, [name]: value }));
  };

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="electric-loader h-12 w-12 rounded-full" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <div className="electric-icon h-10 w-10 rounded-xl flex items-center justify-center">
          <Activity className="h-5 w-5 text-[#9bc3ff]" />
        </div>
        <div>
          <h2 className="text-lg font-semibold text-white">Heartbeat</h2>
          <p className="text-sm text-[#8aa8df]">
            Cron jobs that run in the background. Enable/disable or change schedules.
          </p>
        </div>
      </div>

      {!hasWriteAccess && (
        <div className="electric-card p-4 border-amber-500/30 bg-amber-950/20">
          <div className="flex items-start gap-3">
            <AlertTriangle className="h-5 w-5 text-amber-400 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-amber-200">
                Write access required
              </p>
              <p className="text-sm text-amber-300/80 mt-1">
                Connect your wallet and sign to modify heartbeat schedules.
              </p>
            </div>
          </div>
        </div>
      )}

      {error && (
        <div className="electric-card p-3 border-rose-500/30 bg-rose-950/20 flex items-center gap-2">
          <AlertTriangle className="h-4 w-4 text-rose-400 flex-shrink-0" />
          <span className="text-sm text-rose-300">{error}</span>
        </div>
      )}

      <div className="electric-card overflow-hidden">
        <div className="divide-y divide-[#1a3670]">
          {schedules.length === 0 ? (
            <div className="px-4 py-8 text-center text-sm text-[#8aa8df]">
              No heartbeat schedules. Ensure the agent is running with a heartbeat store.
            </div>
          ) : (
            schedules.map((item) => (
              <div key={item.name} className="px-4 py-3">
                <div className="flex items-center justify-between gap-4">
                  <button
                    type="button"
                    onClick={() =>
                      setExpanded((prev) => ({
                        ...prev,
                        [item.name]: !prev[item.name],
                      }))
                    }
                    className="flex min-w-0 flex-1 items-center gap-2 text-left"
                  >
                    {expanded[item.name] ? (
                      <ChevronUp className="h-4 w-4 shrink-0 text-[#7ea5eb]" />
                    ) : (
                      <ChevronDown className="h-4 w-4 shrink-0 text-[#7ea5eb]" />
                    )}
                    <div className="min-w-0">
                      <p className="font-medium text-white">{item.name}</p>
                      <p className="text-xs text-[#8aa8df] mt-0.5">
                        {item.task} · {item.schedule}
                        {item.tierMinimum && item.tierMinimum !== "dead" && (
                          <> · tier ≥ {item.tierMinimum}</>
                        )}
                      </p>
                    </div>
                  </button>
                  <button
                    type="button"
                    onClick={() => handleToggle(item)}
                    disabled={!hasWriteAccess || !!toggling[item.name]}
                    className={`
                      relative inline-flex h-7 w-12 shrink-0 items-center rounded-full
                      transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-[#4f83ff] focus:ring-offset-2 focus:ring-offset-[#050d1f]
                      disabled:opacity-50 disabled:cursor-not-allowed
                      ${item.enabled ? "bg-[#2f8fff]/60" : "bg-[#1a3670]"}
                    `}
                    role="switch"
                    aria-checked={item.enabled}
                    aria-label={`${item.enabled ? "Disable" : "Enable"} ${item.name}`}
                  >
                    <span
                      className={`
                        inline-block h-5 w-5 transform rounded-full bg-white shadow
                        transition-transform duration-200
                        ${item.enabled ? "translate-x-6" : "translate-x-1"}
                      `}
                    />
                    {toggling[item.name] && (
                      <span className="absolute inset-0 flex items-center justify-center">
                        <Loader2 className="h-4 w-4 animate-spin text-[#9bc3ff]" />
                      </span>
                    )}
                  </button>
                </div>

                {expanded[item.name] && (
                  <div className="mt-4 space-y-3 border-t border-[#1a3670] pt-4">
                    <div className="grid gap-2 text-xs">
                      <div className="flex justify-between text-[#8aa8df]">
                        <span>Last run</span>
                        <span className="text-[#b8d4f0]">
                          {formatDateTime(item.lastRun)}
                        </span>
                      </div>
                      {item.nextRun && (
                        <div className="flex justify-between text-[#8aa8df]">
                          <span>Next run</span>
                          <span className="text-[#b8d4f0]">
                            {formatDateTime(item.nextRun)}
                          </span>
                        </div>
                      )}
                    </div>
                    <div>
                      <label className="mb-1 block text-xs font-medium text-[#8aa8df]">
                        Cron schedule
                      </label>
                      <p className="mb-1 text-xs text-[#6b8fcc]">
                        Standard cron expression (e.g. 0 */6 * * * = every 6 hours)
                      </p>
                      <div className="flex gap-2">
                        <input
                          type="text"
                          value={scheduleEdits[item.name] ?? item.schedule}
                          onChange={(e) =>
                            updateScheduleEdit(item.name, e.target.value)
                          }
                          disabled={!hasWriteAccess}
                          placeholder="0 */6 * * *"
                          className="flex-1 rounded-lg border border-[#29509c] bg-[#071228]/90 px-3 py-2 text-sm text-white placeholder:text-[#6b8fcc] focus:border-[#4f83ff] focus:outline-none disabled:opacity-60"
                        />
                        <button
                          type="button"
                          onClick={() => handleSaveSchedule(item)}
                          disabled={
                            !hasWriteAccess ||
                            !!savingSchedule[item.name] ||
                            (scheduleEdits[item.name] ?? item.schedule) === item.schedule
                          }
                          className="electric-button flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium disabled:opacity-50 shrink-0"
                        >
                          {savingSchedule[item.name] ? (
                            <Loader2 className="h-4 w-4 animate-spin" />
                          ) : (
                            <Save className="h-4 w-4" />
                          )}
                          {savingSchedule[item.name] ? "Saving…" : "Save"}
                        </button>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}

import { ChevronDown, ChevronUp, Loader2, Save } from "lucide-react";
import { formatDateTime } from "@/lib/format";
import type { HeartbeatScheduleItem } from "@/lib/api";
import { CronScheduleInput } from "./CronScheduleInput";
import { isValidCron } from "./cronUtils";

const TOGGLE_CLASSES = `
  relative inline-flex h-7 w-12 shrink-0 items-center rounded-full
  transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-[#4f83ff] focus:ring-offset-2 focus:ring-offset-[#050d1f]
  disabled:opacity-50 disabled:cursor-not-allowed
`;

interface ScheduleItemRowProps {
  item: HeartbeatScheduleItem;
  expanded: boolean;
  scheduleEdit: string;
  toggling: boolean;
  saving: boolean;
  hasWriteAccess: boolean;
  onToggleExpand: () => void;
  onToggleEnabled: () => void;
  onScheduleChange: (value: string) => void;
  onSaveSchedule: () => void;
}

export function ScheduleItemRow({
  item,
  expanded,
  scheduleEdit,
  toggling,
  saving,
  hasWriteAccess,
  onToggleExpand,
  onToggleEnabled,
  onScheduleChange,
  onSaveSchedule,
}: ScheduleItemRowProps) {
  const currentSchedule = scheduleEdit ?? item.schedule;
  const hasScheduleChanged = currentSchedule !== item.schedule;
  const isScheduleValid = isValidCron(currentSchedule);
  const canSave = hasWriteAccess && !saving && hasScheduleChanged && isScheduleValid;

  return (
    <div className="px-4 py-3">
      <div className="flex items-center justify-between gap-4">
        <button
          type="button"
          onClick={onToggleExpand}
          className="flex min-w-0 flex-1 items-center gap-2 text-left"
        >
          {expanded ? (
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
          onClick={onToggleEnabled}
          disabled={!hasWriteAccess || toggling}
          className={`${TOGGLE_CLASSES} ${item.enabled ? "bg-[#2f8fff]/60" : "bg-[#1a3670]"}`}
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
          {toggling && (
            <span className="absolute inset-0 flex items-center justify-center">
              <Loader2 className="h-4 w-4 animate-spin text-[#9bc3ff]" />
            </span>
          )}
        </button>
      </div>

      {expanded && (
        <div className="mt-4 space-y-3 border-t border-[#1a3670] pt-4">
          <div className="grid gap-2 text-xs">
            <div className="flex justify-between text-[#8aa8df]">
              <span>Last run</span>
              <span className="text-[#b8d4f0]">{formatDateTime(item.lastRun)}</span>
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
            <p className="mb-2 text-xs text-[#6b8fcc]">
              Five fields: minute, hour, day of month, month, day of week.{" "}
              <a
                href="https://crontab.guru/"
                target="_blank"
                rel="noopener noreferrer"
                className="text-[#7ea5eb] hover:underline"
              >
                crontab.guru
              </a>{" "}
              for examples.
            </p>
            <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
              <div className="min-w-0 flex-1">
                <CronScheduleInput
                  value={scheduleEdit ?? item.schedule}
                  onChange={onScheduleChange}
                  disabled={!hasWriteAccess}
                />
              </div>
              <button
                type="button"
                onClick={onSaveSchedule}
                disabled={!canSave}
                className="electric-button flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium disabled:opacity-50 shrink-0"
              >
                {saving ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Save className="h-4 w-4" />
                )}
                {saving ? "Saving…" : "Save"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

import { inputBase } from "@/lib/theme";
import { parseCron, buildCron, cronToDescription, isValidCron } from "./cronUtils";

const FIELD_CLASSES = `
  ${inputBase} w-full min-w-0 px-2 py-1.5 text-center text-sm
  font-mono
`;

const LABELS = [
  { key: "minute", label: "minute", placeholder: "*" },
  { key: "hour", label: "hour", placeholder: "*" },
  { key: "day", label: "day", placeholder: "*" },
  { key: "month", label: "month", placeholder: "*" },
  { key: "weekday", label: "weekday", placeholder: "*" },
] as const;

interface CronScheduleInputProps {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
}

export function CronScheduleInput({
  value,
  onChange,
  disabled = false,
}: CronScheduleInputProps) {
  const parts = parseCron(value);
  const [minute, hour, dayOfMonth, month, dayOfWeek] = parts;

  const handleChange = (index: number, newVal: string) => {
    const next = [...parts];
    next[index] = newVal || "*";
    onChange(buildCron(next[0], next[1], next[2], next[3], next[4]));
  };

  const fields = [minute, hour, dayOfMonth, month, dayOfWeek];
  const valid = isValidCron(value);
  const description = valid ? cronToDescription(value) : null;

  return (
    <div className="space-y-2">
      {/* 5-field row - crontab.guru style */}
      <div className="flex flex-wrap items-end gap-2">
        {LABELS.map(({ key, label, placeholder: p }, i) => (
          <div key={key} className="flex min-w-0 flex-1 basis-16 flex-col gap-0.5">
            <label className="text-[10px] font-medium uppercase tracking-wider text-[#6b8fcc]">
              {label}
            </label>
            <input
              type="text"
              value={fields[i]}
              onChange={(e) => handleChange(i, e.target.value)}
              onFocus={(e) => e.target.select()}
              disabled={disabled}
              placeholder={p}
              className={FIELD_CLASSES}
              aria-label={`Cron ${label} field`}
              aria-invalid={!valid && !!value?.trim()}
            />
          </div>
        ))}
      </div>
      {/* Human-readable description or invalid state */}
      {value?.trim() && (
        <p className={`text-xs ${valid ? "text-[#8aa8df]" : "text-amber-400"}`}>
          <span className="text-[#6b8fcc]">→</span>{" "}
          {valid ? description : "Invalid cron expression"}
        </p>
      )}
    </div>
  );
}

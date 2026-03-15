import cronstrue from "cronstrue";

/** Parse a 5-field cron expression into parts. Pads with * if fewer than 5 fields. */
export function parseCron(expr: string): [string, string, string, string, string] {
  const trimmed = expr?.trim() ?? "";
  const parts = trimmed ? trimmed.split(/\s+/) : [];
  const defaults: [string, string, string, string, string] = ["*", "*", "*", "*", "*"];
  for (let i = 0; i < 5; i++) {
    defaults[i] = parts[i] ?? "*";
  }
  return defaults;
}

/** Build a 5-field cron expression from parts. */
export function buildCron(
  minute: string,
  hour: string,
  dayOfMonth: string,
  month: string,
  dayOfWeek: string
): string {
  return [minute, hour, dayOfMonth, month, dayOfWeek].join(" ");
}

const CRONSTRUE_OPTS = {
  throwExceptionOnParseError: true,
  use24HourTimeFormat: false,
  dayOfWeekStartIndexZero: true,
  monthStartIndexZero: false,
} as const;

/** Returns true if the expression is a valid cron schedule. */
export function isValidCron(expr: string): boolean {
  const trimmed = expr?.trim() ?? "";
  if (!trimmed) return false;
  try {
    cronstrue.toString(trimmed, CRONSTRUE_OPTS);
    return true;
  } catch {
    return false;
  }
}

/**
 * Human-readable description using cRonstrue (production-grade logic).
 * Supports @-aliases, L/W/#, ranges, steps, lists, and 5/6/7-field expressions.
 */
export function cronToDescription(expr: string): string {
  const trimmed = expr?.trim() ?? "";
  if (!trimmed) return "";
  try {
    return cronstrue.toString(trimmed, CRONSTRUE_OPTS);
  } catch {
    return trimmed;
  }
}

/** Format ISO date string for display. Returns fallback for invalid/empty input. */
export function formatDateTime(s: string): string {
  if (!s) return "—";
  try {
    const d = new Date(s);
    return isNaN(d.getTime()) ? s : d.toLocaleString();
  } catch {
    return s;
  }
}

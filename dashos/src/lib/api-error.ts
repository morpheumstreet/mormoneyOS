/** Centralized API error handling — DRY replacement for repeated catch blocks */
export function handleApiError(
  error: unknown,
  setError: (msg: string | null) => void,
  fallback: string
): void {
  setError(error instanceof Error ? error.message : fallback);
}

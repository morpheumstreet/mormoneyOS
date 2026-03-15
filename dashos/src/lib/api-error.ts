/** Centralized API error handling — DRY replacement for repeated catch blocks */

/** Extract a user-facing message from an unknown error. */
export function getApiErrorMessage(error: unknown, fallback: string): string {
  return error instanceof Error ? error.message : fallback;
}

export function handleApiError(
  error: unknown,
  setError: (msg: string | null) => void,
  fallback: string
): void {
  setError(getApiErrorMessage(error, fallback));
}

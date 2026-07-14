/**
 * Pull the message out of a backend `{"error":"..."}` envelope, if present.
 * Shared by the API hooks so the envelope is parsed in exactly one place.
 */
export function backendError(error: unknown): string | undefined {
  if (error && typeof error === "object" && "error" in error) {
    const message = (error as Record<string, unknown>).error;
    if (typeof message === "string") return message;
  }
  return undefined;
}

/**
 * A backend error carrying its HTTP status, so a caller can distinguish e.g. a `404`
 * (already-gone/not-owned — SPEC-211 BR-2111, refresh the list silently) from a `400`
 * (validation — show the message inline).
 */
export class ApiError extends Error {
  constructor(
    public readonly status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

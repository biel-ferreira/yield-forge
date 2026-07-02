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

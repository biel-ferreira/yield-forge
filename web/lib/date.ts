// Date formatting at the render edge (SPEC-211). Wire dates are always "YYYY-MM-DD"
// (api/openapi.yaml `format: date`) — this converts to a pt-BR "DD/MM/AAAA" display string
// ONLY here, via plain string manipulation (never a Date object / timezone-sensitive parse).
export function formatDateBR(isoDate: string): string {
  const [year, month, day] = isoDate.split("-");
  return `${day}/${month}/${year}`;
}

/** Today as "YYYY-MM-DD", for a native <input type="date"> min bound. Local calendar date. */
export function todayISO(): string {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, "0");
  const day = String(now.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

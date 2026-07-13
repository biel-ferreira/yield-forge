// The money/rate formatting edge (SPEC-200 FR-2005, BR-2003) — the client analogue of
// internal/platform/money. Money is integer centavos and rates are integer basis points
// EVERYWHERE in the app; these helpers convert to pt-BR display strings ONLY here, at the
// render edge. No float ever represents a balance or a rate — the value math is integer.

/** Integer centavos → "R$ 1.234,56" (pt-BR). e.g. formatCentavos(1234567) === "R$ 12.345,67". */
export function formatCentavos(centavos: number): string {
  const negative = centavos < 0;
  const abs = Math.abs(Math.trunc(centavos));
  const reais = Math.trunc(abs / 100);
  const cents = abs % 100;
  return `${negative ? "-" : ""}R$ ${reais.toLocaleString("pt-BR")},${String(cents).padStart(2, "0")}`;
}

/** Integer basis points → "10,50%" (pt-BR, 2 decimals). For rates: DY, SELIC, yields. */
export function formatBps(bps: number): string {
  const negative = bps < 0;
  const abs = Math.abs(Math.trunc(bps));
  const whole = Math.trunc(abs / 100);
  const frac = abs % 100;
  return `${negative ? "-" : ""}${whole.toLocaleString("pt-BR")},${String(frac).padStart(2, "0")}%`;
}

/** Integer basis points → compact percent for allocation shares: "26%", "16,5%", "0,82%". */
export function formatShareBps(bps: number): string {
  const abs = Math.abs(Math.trunc(bps));
  const whole = Math.trunc(abs / 100);
  const frac = abs % 100;
  const sign = bps < 0 ? "-" : "";
  if (frac === 0) return `${sign}${whole.toLocaleString("pt-BR")}%`;
  const fracStr = String(frac).padStart(2, "0").replace(/0$/, "");
  return `${sign}${whole.toLocaleString("pt-BR")},${fracStr}%`;
}

// The input-side counterpart (SPEC-211 FR-2119, BR-2112): a pt-BR-formatted currency/percentage
// string parsed to an integer, never a float — before it ever reaches a request body. Malformed
// or empty input returns null (never silently coerced to 0); the caller decides how to surface it.
const NUMERIC_PT_BR = /^-?\d{1,3}(\.\d{3})*(,\d+)?$|^-?\d+(,\d+)?$/;

function parsePtBrDecimal(input: string): number | null {
  const trimmed = input.trim();
  if (trimmed === "" || !NUMERIC_PT_BR.test(trimmed)) return null;
  const normalized = trimmed.replace(/\./g, "").replace(",", ".");
  const value = Number(normalized);
  return Number.isFinite(value) ? value : null;
}

/** "1.234,56" → 123456 (integer centavos). Malformed/empty input → null, never coerced to 0. */
export function parseCentavos(input: string): number | null {
  const value = parsePtBrDecimal(input);
  if (value === null) return null;
  return Math.round(value * 100);
}

/** "10,5" → 1050 (integer basis points). Malformed/empty input → null, never coerced to 0. */
export function parseBps(input: string): number | null {
  const value = parsePtBrDecimal(input);
  if (value === null) return null;
  return Math.round(value * 100);
}

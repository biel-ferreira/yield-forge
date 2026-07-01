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

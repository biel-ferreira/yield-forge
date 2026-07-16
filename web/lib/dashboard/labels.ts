// FII sector ↔ pt-BR label mapping (SPEC-212 FR-2124). Unlike SPEC-211's INDEXER_LABELS, this
// is deliberately NOT typed as Record<Sector, string>: api/openapi.yaml declares
// `fii_sectors[].sector` as plain `type: string`, with no `enum:` constraint (unlike
// `indexer_type`) — so the generated TS type is plain `string`, not a closed union (PLAN-212
// D6). A Record<string,string> + a fallback keeps a future/unmapped backend sector value
// legible (its raw value, capitalized) instead of rendering blank or throwing.
export const SECTOR_LABELS: Record<string, string> = {
  logistics: "Logística",
  offices: "Lajes Corporativas",
  shopping: "Shoppings",
  hybrid: "Híbrido",
  paper: "Papel (CRI)",
  // The backend's degradation sector (SPEC-103 FR-1033) for a held FII with no stored quote —
  // labelled distinctly so it reads as a data-quality signal, not a real sector choice.
  other: "Outros / sem cotação",
};

/** A sector's pt-BR label, falling back to its capitalized raw value if unmapped (D6). */
export function sectorLabel(sector: string): string {
  const known = SECTOR_LABELS[sector];
  if (known) return known;
  return sector.length === 0 ? sector : sector[0].toUpperCase() + sector.slice(1);
}

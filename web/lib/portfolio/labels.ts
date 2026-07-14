import type { components } from "@/lib/api/schema";

// Enum ↔ pt-BR label mapping (SPEC-211, mirrors lib/profile/labels.ts). The wire always
// carries the API's enum values; the UI always shows the pt-BR labels. Typed from the
// generated contract, so a new enum value fails the build HERE (a missing Record key)
// rather than silently.
type FixedIncomeReq = components["schemas"]["FixedIncomeRequest"];
type MarketIndicator = components["schemas"]["MarketIndicatorResponse"];

export type Indexer = FixedIncomeReq["indexer_type"];
export type LiquidityType = FixedIncomeReq["liquidity_type"];
export type Indicator = MarketIndicator["indicator"];

export const INDEXER_LABELS: Record<Indexer, string> = {
  prefixado: "Prefixado",
  cdi_percentual: "% do CDI",
  ipca_spread: "IPCA +",
};

export const LIQUIDITY_LABELS: Record<LiquidityType, string> = {
  daily: "Diária",
  at_maturity: "No vencimento",
};

export const INDICATOR_LABELS: Record<Indicator, string> = {
  selic: "SELIC",
  cdi: "CDI",
  ipca: "IPCA",
};

// Stable render order.
export const INDEXERS = Object.keys(INDEXER_LABELS) as Indexer[];
export const LIQUIDITY_TYPES = Object.keys(LIQUIDITY_LABELS) as LiquidityType[];

/** The reference Indicator a given Indexer resolves against, or null for prefixado (no lookup). */
export function referenceIndicator(indexer: Indexer): Indicator | null {
  switch (indexer) {
    case "cdi_percentual":
      return "cdi";
    case "ipca_spread":
      return "ipca";
    case "prefixado":
      return null;
  }
}

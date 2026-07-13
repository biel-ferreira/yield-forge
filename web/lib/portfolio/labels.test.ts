import { describe, it, expect } from "vitest";
import {
  INDEXERS,
  INDEXER_LABELS,
  LIQUIDITY_TYPES,
  LIQUIDITY_LABELS,
  INDICATOR_LABELS,
  referenceIndicator,
} from "@/lib/portfolio/labels";

describe("portfolio labels (SPEC-211/SPEC-109)", () => {
  it("maps every indexer to a non-empty pt-BR label", () => {
    expect(INDEXERS).toEqual(["prefixado", "cdi_percentual", "ipca_spread"]);
    for (const i of INDEXERS) expect(INDEXER_LABELS[i]).toBeTruthy();
    expect(INDEXER_LABELS.cdi_percentual).toBe("% do CDI");
  });

  it("maps every liquidity type to a non-empty pt-BR label", () => {
    expect(LIQUIDITY_TYPES).toEqual(["daily", "at_maturity"]);
    for (const l of LIQUIDITY_TYPES) expect(LIQUIDITY_LABELS[l]).toBeTruthy();
    expect(LIQUIDITY_LABELS.daily).toBe("Diária");
  });

  it("maps every market indicator to a non-empty pt-BR label", () => {
    expect(INDICATOR_LABELS.selic).toBe("SELIC");
    expect(INDICATOR_LABELS.cdi).toBe("CDI");
    expect(INDICATOR_LABELS.ipca).toBe("IPCA");
  });
});

describe("referenceIndicator (SPEC-109 D3 — which indicator an indexer resolves against)", () => {
  it("prefixado has no reference indicator (nothing to resolve)", () => {
    expect(referenceIndicator("prefixado")).toBeNull();
  });
  it("cdi_percentual resolves against CDI", () => {
    expect(referenceIndicator("cdi_percentual")).toBe("cdi");
  });
  it("ipca_spread resolves against IPCA", () => {
    expect(referenceIndicator("ipca_spread")).toBe("ipca");
  });
});

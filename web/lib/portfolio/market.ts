import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api/client";
import type { components } from "@/lib/api/schema";
import type { Indicator } from "@/lib/portfolio/labels";

// The SPEC-006 macro reference rates, exposed read-only by SPEC-109's GET /market/indicators —
// consumed for the live indexer reference display (FR-2120). Reference rates are always read
// from the server, never computed or hardcoded client-side (BR-2117).
export type MarketIndicator = components["schemas"]["MarketIndicatorResponse"];

const MARKET_INDICATORS_KEY = ["market", "indicators"] as const;

/** The latest SELIC/CDI/IPCA readings. An indicator with no ingested value yet is simply absent. */
export function useMarketIndicators() {
  const query = useQuery({
    queryKey: MARKET_INDICATORS_KEY,
    queryFn: async () => {
      const { data } = await api.GET("/market/indicators");
      if (!data) throw new Error("failed to load market indicators");
      return data;
    },
  });
  return {
    indicators: query.data ?? [],
    isLoading: query.isLoading,
    isError: query.isError,
  };
}

/** The list entry for a given indicator, or undefined if it's not present (never ingested/failed). */
export function findIndicator(
  indicators: MarketIndicator[],
  indicator: Indicator,
): MarketIndicator | undefined {
  return indicators.find((entry) => entry.indicator === indicator);
}

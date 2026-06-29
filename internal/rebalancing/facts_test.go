package rebalancing

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/insight"
	"github.com/biel-ferreira/yield-forge/internal/marketdata"
)

func baseFacts() insight.Facts {
	return insight.Facts{
		"current_value_centavos": int64(100_000),
		"allocation_bps":         map[string]int{"fii": 10000}, // 100% FII
		"risk_profile":           "moderate",
	}
}

func sampleUniverse() []marketdata.FIIQuote {
	return []marketdata.FIIQuote{
		{Ticker: marketdata.MustParseTicker("HGLG11"), Sector: marketdata.SectorLogistics, DividendYieldBps: 850, PriceCentavos: 16_000},
		{Ticker: marketdata.MustParseTicker("KNRI11"), Sector: marketdata.SectorHybrid, DividendYieldBps: 900, PriceCentavos: 13_000},
	}
}

func TestAssembleFacts_AugmentsAndComputesSplit(t *testing.T) {
	facts, split := assembleFacts(baseFacts(), mustContribution(t, 100_000), sampleUniverse())

	require.Equal(t, int64(100_000), facts["contribution_centavos"])

	// 100% FII + moderate (50/50) → contribution steered to the under-weight Fixed Income.
	require.Len(t, split, 1)
	require.Equal(t, "fixed_income", string(split[0].Class))
	require.Equal(t, 10000, split[0].SuggestedShareBps)

	// The split is mirrored into the facts as integers.
	splitFact := facts["suggested_split"].([]map[string]any)
	require.Len(t, splitFact, 1)
	require.Equal(t, "fixed_income", splitFact[0]["area"])
	require.Equal(t, 10000, splitFact[0]["suggested_share_bps"])

	// The universe is grounded (ticker/sector/yield) and ordered as provided.
	universe := facts["fii_universe"].([]map[string]any)
	require.Len(t, universe, 2)
	require.Equal(t, "HGLG11", universe[0]["ticker"])
	require.Equal(t, 850, universe[0]["dividend_yield_bps"])
}

func TestAllocationFromFacts_ReconstructsPerClassValue(t *testing.T) {
	base := insight.Facts{
		"current_value_centavos": int64(2_820_000),
		"allocation_bps":         map[string]int{"fii": 6028, "fixed_income": 3972},
	}
	alloc := allocationFromFacts(base)
	require.Equal(t, int64(2_820_000), alloc.TotalCentavos)
	// 60.28% of 2_820_000 = 1_699_896 (half-up).
	require.Equal(t, int64(1_699_896), alloc.ByClass["fii"])
	require.Equal(t, int64(1_120_104), alloc.ByClass["fixed_income"])
}

func TestAssembleFacts_EmptyPortfolioFollowsDirection(t *testing.T) {
	base := insight.Facts{
		"current_value_centavos": int64(0),
		"risk_profile":           "conservative",
	}
	_, split := assembleFacts(base, mustContribution(t, 1_000_000), nil)
	// Empty portfolio + conservative → split follows the 70/30 direction, still reconciles.
	sum := 0
	for _, a := range split {
		sum += a.SuggestedShareBps
	}
	require.Equal(t, 10000, sum)
}

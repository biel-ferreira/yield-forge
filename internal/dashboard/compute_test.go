package dashboard

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/portfolio"
)

func mustFII(t *testing.T, ticker string, qty int, avgPrice int64) portfolio.FIIHolding {
	t.Helper()
	q, err := portfolio.ParseQuantity(qty)
	require.NoError(t, err)
	return portfolio.FIIHolding{Ticker: marketdata.MustParseTicker(ticker), Quantity: q, AveragePriceCentavos: avgPrice}
}

func quote(ticker string, price int64, sector marketdata.Sector, lastDiv int64) marketdata.FIIQuote {
	return marketdata.FIIQuote{
		Ticker: marketdata.MustParseTicker(ticker), PriceCentavos: price, Sector: sector, LastDividendCentavos: lastDiv,
	}
}

func classValue(d Dashboard, c AssetClass) int64 {
	for _, s := range d.Allocation {
		if s.Class == c {
			return s.ValueCentavos
		}
	}
	return -1
}

func TestCompute_MixedPortfolio(t *testing.T) {
	created := time.Date(2025, 6, 26, 0, 0, 0, 0, time.UTC)
	now := created.AddDate(1, 0, 0) // exactly 365 days later

	holdings := portfolio.Holdings{
		FII: []portfolio.FIIHolding{
			mustFII(t, "HGLG11", 100, 15_750), // cost 1_575_000
			mustFII(t, "KNRI11", 50, 14_820),  // cost 741_000
			mustFII(t, "XPLG11", 10, 10_000),  // cost 100_000 — no quote (stale)
		},
		FixedIncome: []portfolio.FixedIncomeHolding{
			{InvestedAmountCentavos: 1_000_000, AnnualRateBps: 1_200, LiquidityType: portfolio.LiquidityAtMaturity, CreatedAt: created},
		},
	}
	quotes := map[marketdata.Ticker]marketdata.FIIQuote{
		marketdata.MustParseTicker("HGLG11"): quote("HGLG11", 16_000, marketdata.SectorLogistics, 110), // current 1_600_000, income 11_000
		marketdata.MustParseTicker("KNRI11"): quote("KNRI11", 15_000, marketdata.SectorHybrid, 90),     // current 750_000, income 4_500
		// XPLG11 deliberately absent → stale, valued at cost basis 100_000
	}

	d := Compute(holdings, quotes, now)

	// Summary.
	require.Equal(t, int64(3_416_000), d.Summary.TotalInvestedCentavos)
	require.Equal(t, int64(3_570_000), d.Summary.CurrentValueCentavos, "the full patrimony")
	require.Equal(t, int64(15_500), d.Summary.MonthlyIncomeCentavos, "stale XPLG11 contributes 0 income")
	require.Equal(t, int64(154_000), d.Summary.GrowthCentavos)
	require.Equal(t, 451, d.Summary.GrowthBps)

	// FI current value = invested + 12% simple interest over 1 year = 1_120_000.
	require.Equal(t, int64(1_120_000), classValue(d, ClassFixedIncome))
	require.Equal(t, int64(2_450_000), classValue(d, ClassFII)) // 1.6M + 0.75M + 0.1M(stale)
	require.Equal(t, int64(0), classValue(d, ClassStocks))
	require.Equal(t, int64(0), classValue(d, ClassETFs))

	// Stale fallback.
	require.Equal(t, []string{"XPLG11"}, d.StaleTickers)

	// Sectors (fixed order, of the FII total): Logistics, Hybrid, Other.
	require.Len(t, d.FIISectors, 3)
	require.Equal(t, marketdata.SectorLogistics, d.FIISectors[0].Sector)
	require.Equal(t, marketdata.SectorHybrid, d.FIISectors[1].Sector)
	require.Equal(t, marketdata.SectorOther, d.FIISectors[2].Sector) // the stale XPLG11

	// Reconciliation (BR-1034): Σ class values = current total; Σ sector values = FII total.
	var classSum, sectorSum int64
	for _, s := range d.Allocation {
		classSum += s.ValueCentavos
	}
	for _, s := range d.FIISectors {
		sectorSum += s.ValueCentavos
	}
	require.Equal(t, d.Summary.CurrentValueCentavos, classSum, "allocation reconciles to the total")
	require.Equal(t, classValue(d, ClassFII), sectorSum, "sectors reconcile to the FII total")
}

func TestCompute_Deterministic(t *testing.T) {
	now := time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)
	holdings := portfolio.Holdings{FII: []portfolio.FIIHolding{mustFII(t, "HGLG11", 100, 15_750)}}
	quotes := map[marketdata.Ticker]marketdata.FIIQuote{
		marketdata.MustParseTicker("HGLG11"): quote("HGLG11", 16_000, marketdata.SectorLogistics, 110),
	}
	require.Equal(t, Compute(holdings, quotes, now), Compute(holdings, quotes, now), "same inputs → same output")
}

func TestCompute_Empty(t *testing.T) {
	d := Compute(portfolio.Holdings{}, nil, time.Now())
	require.Zero(t, d.Summary.CurrentValueCentavos)
	require.Zero(t, d.Summary.GrowthBps, "no divide-by-zero on an empty portfolio")
	require.Empty(t, d.FIISectors)
	require.Empty(t, d.StaleTickers)
	require.Len(t, d.Allocation, 4, "all four classes always present, at 0")
}

func TestCompute_ZeroTotalNoPanic(t *testing.T) {
	// A holding with 0 average price and no quote → all values 0, shares 0, no divide-by-zero.
	holdings := portfolio.Holdings{FII: []portfolio.FIIHolding{mustFII(t, "HGLG11", 100, 0)}}
	d := Compute(holdings, nil, time.Now())
	require.Zero(t, d.Summary.CurrentValueCentavos)
	for _, s := range d.Allocation {
		require.Zero(t, s.ShareBps)
	}
}

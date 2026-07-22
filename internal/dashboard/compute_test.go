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

func classSlice(d Dashboard, c AssetClass) ClassSlice {
	for _, s := range d.Allocation {
		if s.Class == c {
			return s
		}
	}
	return ClassSlice{}
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
			// EffectiveAnnualRateBps (SPEC-109) is what Compute reads for accrual; here it equals
			// the raw stored AnnualRateBps (a prefixado-shaped fixture — ListHoldings resolves
			// this field in production, Compute's own tests set it directly). Never reconciled
			// (SPEC-110): TotalContributedCentavos == InvestedAmountCentavos and LastReconciledAt
			// == CreatedAt — exactly what the migration backfills for a pre-SPEC-110 row (BR-1103),
			// so this fixture doubles as the byte-for-byte regression proof.
			{
				InvestedAmountCentavos: 1_000_000, TotalContributedCentavos: 1_000_000,
				AnnualRateBps: 1_200, EffectiveAnnualRateBps: 1_200, LiquidityType: portfolio.LiquidityAtMaturity,
				LastReconciledAt: created, CreatedAt: created,
			},
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

func TestCompute_Loss(t *testing.T) {
	// Bought at 200,00; the market dropped to 150,00 → a loss, with negative growth bps.
	holdings := portfolio.Holdings{FII: []portfolio.FIIHolding{mustFII(t, "HGLG11", 10, 20_000)}} // cost 200_000
	quotes := map[marketdata.Ticker]marketdata.FIIQuote{
		marketdata.MustParseTicker("HGLG11"): quote("HGLG11", 15_000, marketdata.SectorLogistics, 0), // current 150_000
	}
	d := Compute(holdings, quotes, time.Now())
	require.Equal(t, int64(-50_000), d.Summary.GrowthCentavos)
	require.Equal(t, -2_500, d.Summary.GrowthBps, "a 25% loss → -2500 bps")
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

// TestCompute_PerClassGrowth proves SPEC-110 FR-1104: per-class growth is exposed on
// ClassSlice and Σ(per-class growth) reconciles to the blended Summary.GrowthCentavos.
func TestCompute_PerClassGrowth(t *testing.T) {
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	holdings := portfolio.Holdings{
		FII: []portfolio.FIIHolding{mustFII(t, "HGLG11", 100, 15_000)}, // cost 1_500_000
		FixedIncome: []portfolio.FixedIncomeHolding{{
			InvestedAmountCentavos: 1_000_000, TotalContributedCentavos: 1_000_000,
			EffectiveAnnualRateBps: 1_200, LiquidityType: portfolio.LiquidityDaily,
			LastReconciledAt: now.AddDate(0, 0, -365), CreatedAt: now.AddDate(0, 0, -365),
		}},
	}
	quotes := map[marketdata.Ticker]marketdata.FIIQuote{
		marketdata.MustParseTicker("HGLG11"): quote("HGLG11", 16_000, marketdata.SectorLogistics, 0), // current 1_600_000
	}
	d := Compute(holdings, quotes, now)

	var classGrowthSum int64
	for _, s := range d.Allocation {
		classGrowthSum += s.GrowthCentavos
	}
	require.Equal(t, d.Summary.GrowthCentavos, classGrowthSum, "per-class growth reconciles to the blended total")
	require.Equal(t, int64(100_000), classSlice(d, ClassFII).GrowthCentavos, "1_600_000 - 1_500_000")
	require.Equal(t, int64(120_000), classSlice(d, ClassFixedIncome).GrowthCentavos, "1 year @ 12% simple interest")
}

// TestCompute_FIGrowthUsesTotalContributedNotInvested proves SPEC-110 D2: once a holding has
// been reconciled, InvestedAmountCentavos includes confirmed interest — using it as cost basis
// would hide that interest as growth. TotalContributedCentavos (unaffected by interest) is the
// correct basis.
func TestCompute_FIGrowthUsesTotalContributedNotInvested(t *testing.T) {
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	// Already reconciled once: invested grew by confirmed interest, contributed did not.
	h := portfolio.FixedIncomeHolding{
		InvestedAmountCentavos: 1_010_000, TotalContributedCentavos: 1_000_000,
		EffectiveAnnualRateBps: 0, LastReconciledAt: now, CreatedAt: now.AddDate(0, -1, 0),
	}
	d := Compute(portfolio.Holdings{FixedIncome: []portfolio.FixedIncomeHolding{h}}, nil, now)

	fi := classSlice(d, ClassFixedIncome)
	require.Equal(t, int64(1_010_000), fi.ValueCentavos, "no further accrual: LastReconciledAt == now")
	require.Equal(t, int64(1_000_000), fi.InvestedCentavos, "cost basis is TotalContributedCentavos")
	require.Equal(t, int64(10_000), fi.GrowthCentavos, "the already-confirmed interest still counts as growth")
}

// TestCompute_ReconciliationDueAndNeedsAttention proves SPEC-110 FR-1105: NeedsAttention is true
// when either a stale FII or a due fixed-income holding is present, and ReconciliationDue is read
// directly off the holding (already resolved by portfolio.Service in production), never
// recomputed here.
func TestCompute_ReconciliationDueAndNeedsAttention(t *testing.T) {
	now := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)

	t.Run("a due FI holding sets NeedsAttention", func(t *testing.T) {
		due := portfolio.FixedIncomeHolding{
			Name: "CDB Antigo", InvestedAmountCentavos: 1_000, TotalContributedCentavos: 1_000,
			LastReconciledAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), ReconciliationDue: true,
		}
		d := Compute(portfolio.Holdings{FixedIncome: []portfolio.FixedIncomeHolding{due}}, nil, now)
		require.Equal(t, []string{"CDB Antigo"}, d.FixedIncomeReconciliationDue)
		require.True(t, d.NeedsAttention)
	})

	t.Run("no due holdings and no stale FIIs -> NeedsAttention false", func(t *testing.T) {
		notDue := portfolio.FixedIncomeHolding{
			Name: "CDB Novo", InvestedAmountCentavos: 1_000, TotalContributedCentavos: 1_000,
			LastReconciledAt: now, ReconciliationDue: false,
		}
		d := Compute(portfolio.Holdings{FixedIncome: []portfolio.FixedIncomeHolding{notDue}}, nil, now)
		require.Empty(t, d.FixedIncomeReconciliationDue)
		require.False(t, d.NeedsAttention)
	})

	t.Run("a stale FII alone also sets NeedsAttention", func(t *testing.T) {
		d := Compute(portfolio.Holdings{FII: []portfolio.FIIHolding{mustFII(t, "HGLG11", 1, 100)}}, nil, now)
		require.NotEmpty(t, d.StaleTickers)
		require.True(t, d.NeedsAttention)
	})
}

// TestCompute_PerHoldingBreakdown proves SPEC-110 FR-1109: per-ticker/per-holding value entries
// reconcile to their class total, a stale FII still appears (at cost basis), and order matches
// the input order (never re-sorted).
func TestCompute_PerHoldingBreakdown(t *testing.T) {
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	holdings := portfolio.Holdings{
		FII: []portfolio.FIIHolding{
			mustFII(t, "HGLG11", 100, 15_000), // cost 1_500_000, quoted
			mustFII(t, "XPLG11", 50, 10_000),  // cost 500_000, no quote (stale)
		},
		FixedIncome: []portfolio.FixedIncomeHolding{
			{ID: "fi-1", Name: "CDB A", InvestedAmountCentavos: 1_000_000, TotalContributedCentavos: 1_000_000, LastReconciledAt: now, CreatedAt: now},
			{ID: "fi-2", Name: "CDB B", InvestedAmountCentavos: 500_000, TotalContributedCentavos: 500_000, LastReconciledAt: now, CreatedAt: now},
		},
	}
	quotes := map[marketdata.Ticker]marketdata.FIIQuote{
		marketdata.MustParseTicker("HGLG11"): quote("HGLG11", 16_000, marketdata.SectorLogistics, 0), // current 1_600_000
	}
	d := Compute(holdings, quotes, now)

	require.Len(t, d.FIIHoldings, 2)
	require.Equal(t, "HGLG11", d.FIIHoldings[0].Ticker.String(), "preserves input order")
	require.Equal(t, "XPLG11", d.FIIHoldings[1].Ticker.String())
	require.Equal(t, int64(500_000), d.FIIHoldings[1].ValueCentavos, "stale FII still appears, at cost basis")

	var fiiSum int64
	for _, s := range d.FIIHoldings {
		fiiSum += s.ValueCentavos
	}
	require.Equal(t, classValue(d, ClassFII), fiiSum, "FII holdings reconcile to the class total")

	require.Len(t, d.FixedIncomeHoldings, 2)
	require.Equal(t, "fi-1", d.FixedIncomeHoldings[0].ID, "preserves input order")
	var fiSum int64
	for _, s := range d.FixedIncomeHoldings {
		fiSum += s.ValueCentavos
	}
	require.Equal(t, classValue(d, ClassFixedIncome), fiSum, "FI holdings reconcile to the class total")
}

func TestCompute_Empty_PerHoldingListsAreEmpty(t *testing.T) {
	d := Compute(portfolio.Holdings{}, nil, time.Now())
	require.Empty(t, d.FIIHoldings)
	require.Empty(t, d.FixedIncomeHoldings)
	require.False(t, d.NeedsAttention)
}

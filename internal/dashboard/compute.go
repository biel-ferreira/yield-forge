package dashboard

import (
	"time"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/platform/money"
	"github.com/biel-ferreira/yield-forge/internal/portfolio"
)

// sectorOrder fixes the iteration order of the FII sector breakdown so the output is
// deterministic (a map would not be — BR-1031).
var sectorOrder = []marketdata.Sector{
	marketdata.SectorLogistics, marketdata.SectorOffices, marketdata.SectorShopping,
	marketdata.SectorHybrid, marketdata.SectorPaper, marketdata.SectorOther,
}

// Compute derives the Dashboard from a user's holdings and the quotes fetched for their FIIs,
// as of now. It is a PURE, side-effect-free, deterministic function (SPEC-103 BR-1031): the
// same inputs always yield the same figures. All money is int64 centavos and all shares
// integer basis points (half-up via money), and the breakdowns reconcile against the totals
// (BR-1034). A held FII absent from quotes is valued at cost basis and reported stale (FR-1036).
//
// SPEC-110 extends this: FII and Fixed Income growth are tracked per class (not just blended into
// Summary.GrowthCentavos), the FI accrual anchors off LastReconciledAt (the reconciliation clock)
// instead of CreatedAt, FI growth uses TotalContributedCentavos as its cost basis (not
// InvestedAmountCentavos, which now includes confirmed interest — BR-1101), and each FII/FI
// holding gets its own value+growth+share entry (FR-1109). ReconciliationDue is read directly off
// each FixedIncomeHolding (already resolved by portfolio.Service.withEffectiveRates) rather than
// recomputed here, so there is exactly one now-dependent code path for that fact.
func Compute(holdings portfolio.Holdings, quotes map[marketdata.Ticker]marketdata.FIIQuote, now time.Time) Dashboard {
	var (
		fiiInvested, fiInvested, fiContributed int64
		fiiCurrent, fiCurrent                  int64
		monthlyIncome                          int64
		stale, reconciliationDue               []string
		sectorValue                            = map[marketdata.Sector]int64{}
		fiiHoldings                            []FIIHoldingSlice
		fiHoldings                             []FixedIncomeHoldingSlice
	)

	for _, h := range holdings.FII {
		qty := int64(h.Quantity.Value())
		invested := h.AveragePriceCentavos * qty
		fiiInvested += invested

		var current int64
		var sector marketdata.Sector
		if q, ok := quotes[h.Ticker]; ok {
			current = q.PriceCentavos * qty
			sector = q.Sector
			monthlyIncome += q.LastDividendCentavos * qty
		} else {
			current = invested // cost-basis fallback for a missing/stale quote
			sector = marketdata.SectorOther
			stale = append(stale, h.Ticker.String())
		}
		fiiCurrent += current
		sectorValue[sector] += current
		fiiHoldings = append(fiiHoldings, FIIHoldingSlice{Ticker: h.Ticker, ValueCentavos: current})
	}

	for _, h := range holdings.FixedIncome {
		fiInvested += h.InvestedAmountCentavos
		fiContributed += h.TotalContributedCentavos
		// EffectiveAnnualRateBps (SPEC-109); LastReconciledAt (SPEC-110), not CreatedAt — the
		// accrual clock resets on every reconciliation/balance edit.
		accrued := money.AccrueSimpleInterest(h.InvestedAmountCentavos, h.EffectiveAnnualRateBps, daysBetween(h.LastReconciledAt, now))
		current := h.InvestedAmountCentavos + accrued
		fiCurrent += current
		fiHoldings = append(fiHoldings, FixedIncomeHoldingSlice{
			ID: h.ID, Name: h.Name, ValueCentavos: current,
			GrowthCentavos: current - h.TotalContributedCentavos,
		})
		if h.ReconciliationDue {
			reconciliationDue = append(reconciliationDue, h.Name)
		}
	}

	for i := range fiiHoldings {
		fiiHoldings[i].ShareBps = money.ShareBps(fiiHoldings[i].ValueCentavos, fiiCurrent)
	}
	for i := range fiHoldings {
		fiHoldings[i].ShareBps = money.ShareBps(fiHoldings[i].ValueCentavos, fiCurrent)
	}

	totalInvested := fiiInvested + fiInvested
	totalCurrent := fiiCurrent + fiCurrent
	growth := totalCurrent - totalInvested

	return Dashboard{
		Summary: Summary{
			TotalInvestedCentavos: totalInvested,
			CurrentValueCentavos:  totalCurrent,
			MonthlyIncomeCentavos: monthlyIncome,
			GrowthCentavos:        growth,
			GrowthBps:             money.ShareBps(growth, totalInvested),
		},
		Allocation:                   allocation(fiiCurrent, fiiInvested, fiCurrent, fiContributed, totalCurrent),
		FIISectors:                   sectors(sectorValue, fiiCurrent),
		StaleTickers:                 stale,
		FixedIncomeReconciliationDue: reconciliationDue,
		NeedsAttention:               len(stale) > 0 || len(reconciliationDue) > 0,
		FIIHoldings:                  fiiHoldings,
		FixedIncomeHoldings:          fiHoldings,
	}
}

// allocation returns the four asset-class slices (Stocks/ETFs are 0 in the MVP, FR-1032).
// FII growth is fiiCurrent-fiiInvested (cost basis unchanged, average price × quantity); Fixed
// Income growth is fiCurrent-fiContributed, using TotalContributedCentavos as cost basis — not
// InvestedAmountCentavos — so a reconciled contribution never shows up as growth (SPEC-110
// FR-1104/D2).
func allocation(fiiCurrent, fiiInvested, fiCurrent, fiContributed, total int64) []ClassSlice {
	fiiGrowth := fiiCurrent - fiiInvested
	fiGrowth := fiCurrent - fiContributed
	return []ClassSlice{
		{
			Class: ClassFII, ValueCentavos: fiiCurrent, ShareBps: money.ShareBps(fiiCurrent, total),
			InvestedCentavos: fiiInvested, GrowthCentavos: fiiGrowth, GrowthBps: money.ShareBps(fiiGrowth, fiiInvested),
		},
		{
			Class: ClassFixedIncome, ValueCentavos: fiCurrent, ShareBps: money.ShareBps(fiCurrent, total),
			InvestedCentavos: fiContributed, GrowthCentavos: fiGrowth, GrowthBps: money.ShareBps(fiGrowth, fiContributed),
		},
		{Class: ClassStocks, ValueCentavos: 0, ShareBps: 0},
		{Class: ClassETFs, ValueCentavos: 0, ShareBps: 0},
	}
}

// sectors returns the FII sector breakdown in a fixed order, including only sectors that hold
// value, each as a share of the FII total (FR-1033).
func sectors(value map[marketdata.Sector]int64, fiiTotal int64) []SectorSlice {
	var out []SectorSlice
	for _, s := range sectorOrder {
		if v, ok := value[s]; ok {
			out = append(out, SectorSlice{Sector: s, ValueCentavos: v, ShareBps: money.ShareBps(v, fiiTotal)})
		}
	}
	return out
}

// daysBetween returns the whole days elapsed from `from` to `to`, floored at 0.
func daysBetween(from, to time.Time) int {
	d := to.Sub(from)
	if d < 0 {
		return 0
	}
	return int(d / (24 * time.Hour))
}

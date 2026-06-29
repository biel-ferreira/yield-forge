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
func Compute(holdings portfolio.Holdings, quotes map[marketdata.Ticker]marketdata.FIIQuote, now time.Time) Dashboard {
	var (
		totalInvested int64
		fiiCurrent    int64
		fiCurrent     int64
		monthlyIncome int64
		stale         []string
		sectorValue   = map[marketdata.Sector]int64{}
	)

	for _, h := range holdings.FII {
		qty := int64(h.Quantity.Value())
		invested := h.AveragePriceCentavos * qty
		totalInvested += invested

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
	}

	for _, h := range holdings.FixedIncome {
		totalInvested += h.InvestedAmountCentavos
		accrued := money.AccrueSimpleInterest(h.InvestedAmountCentavos, h.AnnualRateBps, daysBetween(h.CreatedAt, now))
		fiCurrent += h.InvestedAmountCentavos + accrued
	}

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
		Allocation:   allocation(fiiCurrent, fiCurrent, totalCurrent),
		FIISectors:   sectors(sectorValue, fiiCurrent),
		StaleTickers: stale,
	}
}

// allocation returns the four asset-class slices (Stocks/ETFs are 0 in the MVP, FR-1032).
func allocation(fii, fixedIncome, total int64) []ClassSlice {
	return []ClassSlice{
		{Class: ClassFII, ValueCentavos: fii, ShareBps: money.ShareBps(fii, total)},
		{Class: ClassFixedIncome, ValueCentavos: fixedIncome, ShareBps: money.ShareBps(fixedIncome, total)},
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

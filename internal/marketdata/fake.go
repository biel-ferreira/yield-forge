package marketdata

import (
	"context"
	"time"
)

// Fake is a deterministic MarketDataProvider for tests, CI, and offline/dev — it is the
// default MARKETDATA_PROVIDER so the zero-config app never hits the network (SPEC-006
// FR-611). It returns fixed, valid data and never errors.
type Fake struct {
	// At, when set, stamps ObservedAt/FetchedAt/ReferenceDate; otherwise a fixed instant
	// is used so output is reproducible without wiring a Clock.
	At time.Time
}

var _ MarketDataProvider = Fake{}

func (f Fake) stamp() time.Time {
	if f.At.IsZero() {
		return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return f.At.UTC()
}

// FetchFIIQuotes returns one fixed, valid quote per requested ticker.
func (f Fake) FetchFIIQuotes(_ context.Context, tickers []Ticker) (map[Ticker]FIIQuote, error) {
	now := f.stamp()
	lastDiv := now.AddDate(0, -1, 0)
	out := make(map[Ticker]FIIQuote, len(tickers))
	for _, t := range tickers {
		d := lastDiv
		out[t] = FIIQuote{
			Ticker:               t,
			PriceCentavos:        10_000, // R$100,00
			DividendYieldBps:     800,    // 8,00%
			PVPBps:               9_500,  // P/VP 0,95
			Sector:               SectorLogistics,
			LastDividendCentavos: 100, // R$1,00
			LastDividendDate:     &d,
			Source:               "fake",
			ObservedAt:           now,
			FetchedAt:            now,
		}
	}
	return out, nil
}

// FetchMacroIndicator returns a fixed, valid observation for the requested indicator.
func (f Fake) FetchMacroIndicator(_ context.Context, ind Indicator) (MacroIndicator, error) {
	now := f.stamp()
	value, unit := int64(1_050), UnitBps // 10.50% policy rate (1% = 100 bps, like the real BCB adapter)
	if ind == IndicatorIFIX {
		value, unit = 320_000, UnitPoints // ~3.200,00 index points
	}
	return MacroIndicator{
		Indicator:     ind,
		Value:         value,
		Unit:          unit,
		ReferenceDate: now,
		Source:        "fake",
		FetchedAt:     now,
	}, nil
}

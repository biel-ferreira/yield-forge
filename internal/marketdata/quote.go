package marketdata

import "time"

// FIIQuote is the current market snapshot for one FII (SPEC-006 FR-006). Money is int64
// centavos and rates are integer basis points — never float (BR-604). It carries no
// user identity: market data is global reference data (BR-603).
type FIIQuote struct {
	Ticker               Ticker
	PriceCentavos        int64
	DividendYieldBps     int
	PVPBps               int // price/book ratio ×10000 (e.g. 0.95 -> 9500)
	Sector               Sector
	LastDividendCentavos int64
	LastDividendDate     *time.Time // nil when the provider did not supply one
	Source               string     // provider id that produced the row
	ObservedAt           time.Time  // source as-of instant (UTC)
	FetchedAt            time.Time  // when ingested (UTC)
}

// StaleAfter reports whether the quote is older than ttl relative to now, using FetchedAt
// (SPEC-006 FR-606). The Clock-supplied now keeps staleness deterministic in tests.
func (q FIIQuote) StaleAfter(now time.Time, ttl time.Duration) bool {
	return now.Sub(q.FetchedAt) > ttl
}

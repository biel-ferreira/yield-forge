package marketdata

import "time"

// MacroIndicator is one observation of a macro series (SPEC-006 FR-007). Value is an
// integer in Unit (rates in basis points, IFIX in points) — never float (BR-604). Stored
// as a time series keyed by (Indicator, ReferenceDate); a GetLatest read returns the most
// recent ReferenceDate.
type MacroIndicator struct {
	Indicator     Indicator
	Value         int64
	Unit          Unit
	ReferenceDate time.Time // the date the source attributes the value to (UTC, date-only)
	Source        string
	FetchedAt     time.Time // when ingested (UTC)
}

// StaleAfter reports whether the observation is older than ttl relative to now, using
// FetchedAt (SPEC-006 FR-606).
func (m MacroIndicator) StaleAfter(now time.Time, ttl time.Duration) bool {
	return now.Sub(m.FetchedAt) > ttl
}

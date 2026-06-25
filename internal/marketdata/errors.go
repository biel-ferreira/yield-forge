package marketdata

import "errors"

var (
	// ErrFIIQuoteNotFound is returned by a repository read when no quote exists for a ticker.
	ErrFIIQuoteNotFound = errors.New("fii quote not found")
	// ErrMacroNotFound is returned by a repository read when no observation exists for an indicator.
	ErrMacroNotFound = errors.New("macro indicator not found")
	// ErrProviderUnavailable marks a provider failure (outage, rate-limit, malformed/changed
	// payload). The worker degrades on it — last-known-good data is preserved (SPEC-006
	// BR-602, FR-610). Adapters wrap their cause with %w onto this sentinel.
	ErrProviderUnavailable = errors.New("market data provider unavailable")
)

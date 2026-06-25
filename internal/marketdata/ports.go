package marketdata

import "context"

// MarketDataProvider reads current data from an external source — the FR-018 seam that keeps
// the provider swappable by config (BR-601). FII reads are batched because the MVP source
// (Fundamentus) returns every FII in one request; a ticker absent from the returned map is
// a per-item miss the worker treats as "keep last-known-good" (BR-602). On a provider
// failure, methods return a wrapped ErrProviderUnavailable.
type MarketDataProvider interface {
	FetchFIIQuotes(ctx context.Context, tickers []Ticker) (map[Ticker]FIIQuote, error)
	FetchMacroIndicator(ctx context.Context, ind Indicator) (MacroIndicator, error)
}

// FIIQuoteRepository persists the current snapshot per ticker. Writes are idempotent upserts
// keyed by ticker; a failed/empty fetch must never reach here (BR-602). There is no user
// scoping (BR-603).
type FIIQuoteRepository interface {
	UpsertFIIQuote(ctx context.Context, q FIIQuote) error
	GetFIIQuoteByTicker(ctx context.Context, t Ticker) (FIIQuote, error) // ErrFIIQuoteNotFound when absent
}

// MacroRepository persists the macro time series. Upserts are idempotent on
// (Indicator, ReferenceDate); GetLatest returns the newest observation per indicator.
type MacroRepository interface {
	UpsertMacroIndicator(ctx context.Context, m MacroIndicator) error
	GetLatestMacroIndicator(ctx context.Context, ind Indicator) (MacroIndicator, error) // ErrMacroNotFound when absent
}

// TickerSource supplies the FII tickers to refresh each run. The MVP implementation reads a
// configured watchlist; a holdings-backed source arrives with SPEC-102.
type TickerSource interface {
	Tickers(ctx context.Context) ([]Ticker, error)
}

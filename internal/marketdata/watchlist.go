package marketdata

import "context"

// Watchlist is a static TickerSource backed by a configured list of tickers. Since SPEC-007,
// holdings-derived tickers are the primary source (internal/marketdata/ingest's holdingsSource);
// Watchlist now serves as an optional seed unioned with them — handy for seeding local/dev/CI
// before any holdings exist. It validates every ticker up front, so a typo in
// MARKETDATA_WATCHLIST fails fast at startup rather than silently each run.
type Watchlist struct {
	tickers []Ticker
}

var _ TickerSource = Watchlist{}

// NewWatchlist parses raw ticker strings into a Watchlist, returning an error (wrapping
// ErrInvalidTicker) if any entry is not a valid B3 ticker.
func NewWatchlist(raw []string) (Watchlist, error) {
	tickers := make([]Ticker, 0, len(raw))
	for _, s := range raw {
		t, err := ParseTicker(s)
		if err != nil {
			return Watchlist{}, err
		}
		tickers = append(tickers, t)
	}
	return Watchlist{tickers: tickers}, nil
}

// Tickers returns the configured tickers.
func (w Watchlist) Tickers(context.Context) ([]Ticker, error) { return w.tickers, nil }

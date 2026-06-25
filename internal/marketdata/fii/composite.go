// Package fii composes the FII data sources into a single FetchFIIQuotes (SPEC-006 D4):
// fundamentals (price/DY/P-VP/segment, from Fundamentus) enriched with last-dividend (from
// Yahoo). It depends on small consumer-defined interfaces, not the concrete adapters, so
// either source can be swapped behind the marketdata port (BR-601).
package fii

import (
	"context"
	"time"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
)

// Fundamentals supplies the bulk FII fundamentals (Fundamentus).
type Fundamentals interface {
	FetchFIIQuotes(ctx context.Context, tickers []marketdata.Ticker) (map[marketdata.Ticker]marketdata.FIIQuote, error)
}

// LastDividends supplies a ticker's most recent distribution (Yahoo). It is best-effort.
type LastDividends interface {
	FetchLastDividend(ctx context.Context, t marketdata.Ticker) (centavos int64, date *time.Time, err error)
}

// Composite merges fundamentals with last-dividend. It satisfies the FII half of the
// marketdata.MarketDataProvider port.
type Composite struct {
	fundamentals Fundamentals
	dividends    LastDividends
}

// New returns a Composite over the two sources.
func New(fundamentals Fundamentals, dividends LastDividends) Composite {
	return Composite{fundamentals: fundamentals, dividends: dividends}
}

// FetchFIIQuotes fetches fundamentals (a hard dependency — its failure degrades the whole
// batch, preserving last-known-good per BR-602) then enriches each quote with its last
// dividend. The dividend lookup is best-effort: a failure or absence leaves last-dividend
// empty (a partial quote), never failing the row (SPEC-006 FR-602 edge cases).
func (c Composite) FetchFIIQuotes(ctx context.Context, tickers []marketdata.Ticker) (map[marketdata.Ticker]marketdata.FIIQuote, error) {
	quotes, err := c.fundamentals.FetchFIIQuotes(ctx, tickers)
	if err != nil {
		return nil, err
	}
	for t, q := range quotes {
		cents, date, derr := c.dividends.FetchLastDividend(ctx, t)
		if derr == nil && cents > 0 {
			q.LastDividendCentavos = cents
			q.LastDividendDate = date
			quotes[t] = q
		}
	}
	return quotes, nil
}

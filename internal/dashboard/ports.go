package dashboard

import (
	"context"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/portfolio"
)

// HoldingsReader supplies the caller's holdings (SPEC-103 FR-1035). It is a consumer-defined
// interface (accept interfaces) satisfied by *portfolio.Service via its Reader port — the
// dashboard depends on the published read seam, not on portfolio's internals.
type HoldingsReader interface {
	ListHoldings(ctx context.Context, userID string) (portfolio.Holdings, error)
}

// QuoteSource supplies the current market quote for an FII ticker, returning
// marketdata.ErrFIIQuoteNotFound when none is stored (the dashboard then falls back to cost
// basis, FR-1036). Satisfied by the marketdata Postgres quote repository.
type QuoteSource interface {
	GetFIIQuoteByTicker(ctx context.Context, t marketdata.Ticker) (marketdata.FIIQuote, error)
}

package dashboard

import (
	"context"
	"errors"
	"fmt"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/platform/clock"
)

// Service computes a user's dashboard (SPEC-103 FR-1035). It depends only on the consumer
// ports and the Clock, so it is pure application logic — unit-testable with hand-written
// fakes, no DB or HTTP. The userID is always supplied by the caller from the authenticated
// context (BR-1033).
type Service struct {
	holdings HoldingsReader
	quotes   QuoteSource
	clock    clock.Clock
}

// NewService builds a Service over the holdings reader, quote source, and clock.
func NewService(holdings HoldingsReader, quotes QuoteSource, clk clock.Clock) *Service {
	return &Service{holdings: holdings, quotes: quotes, clock: clk}
}

// GetDashboard reads the caller's holdings, fetches the current quote for each distinct held
// FII, and computes the dashboard as of the Clock's now. A FII with no stored quote
// (ErrFIIQuoteNotFound) is omitted from the quote map — Compute then values it at cost basis
// and reports it stale (FR-1036). Any other read error is surfaced.
func (s *Service) GetDashboard(ctx context.Context, userID string) (Dashboard, error) {
	holdings, err := s.holdings.ListHoldings(ctx, userID)
	if err != nil {
		return Dashboard{}, fmt.Errorf("get dashboard: %w", err)
	}

	quotes := make(map[marketdata.Ticker]marketdata.FIIQuote)
	fetched := make(map[marketdata.Ticker]bool) // fetch once per distinct ticker (a user may hold one twice)
	for _, h := range holdings.FII {
		if fetched[h.Ticker] {
			continue
		}
		fetched[h.Ticker] = true

		q, err := s.quotes.GetFIIQuoteByTicker(ctx, h.Ticker)
		if errors.Is(err, marketdata.ErrFIIQuoteNotFound) {
			continue // stale — Compute falls back to cost basis
		}
		if err != nil {
			return Dashboard{}, fmt.Errorf("get dashboard: quote %s: %w", h.Ticker.String(), err)
		}
		quotes[h.Ticker] = q
	}

	return Compute(holdings, quotes, s.clock.Now()), nil
}

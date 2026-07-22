package ingest

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
)

// tickerReader is the narrow slice of portfolio.SystemReader this package needs (accept
// interfaces, consumer-defined — mirrors the existing fiiFetcher/macroFetcher pattern below).
// A portfolio/postgres.Repository satisfies it structurally; this file never imports the
// portfolio package by name, keeping the marketdata core's own boundary unaffected — only the
// ingestion composition edge (this package) reaches across (SPEC-007 BR-072).
type tickerReader interface {
	DistinctFIITickers(ctx context.Context) ([]string, error)
}

// holdingsSource is a marketdata.TickerSource backed by users' actual FII holdings (SPEC-007
// FR-072) — the primary ticker source, replacing the static MARKETDATA_WATCHLIST.
type holdingsSource struct {
	reader tickerReader
	logger *slog.Logger
}

var _ marketdata.TickerSource = holdingsSource{}

// newHoldingsSource returns a holdingsSource reading distinct tickers via reader.
func newHoldingsSource(reader tickerReader, logger *slog.Logger) holdingsSource {
	return holdingsSource{reader: reader, logger: logger}
}

// Tickers returns every distinct held FII ticker, parsed via marketdata.ParseTicker. A
// malformed stored ticker is skipped and logged, never aborting the call (SPEC-007 BR-075) —
// holdings already validate their ticker on write (SPEC-102), so this is a defensive guard,
// not an expected path.
func (h holdingsSource) Tickers(ctx context.Context) ([]marketdata.Ticker, error) {
	raw, err := h.reader.DistinctFIITickers(ctx)
	if err != nil {
		return nil, fmt.Errorf("holdings ticker source: %w", err)
	}
	tickers := make([]marketdata.Ticker, 0, len(raw))
	for _, s := range raw {
		t, err := marketdata.ParseTicker(s)
		if err != nil {
			h.logger.Warn("market data: skipping malformed stored ticker",
				slog.String("ticker", s), slog.String("error", err.Error()))
			continue
		}
		tickers = append(tickers, t)
	}
	return tickers, nil
}

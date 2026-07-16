package ingest

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
)

// unionSource composes N marketdata.TickerSources into the deduplicated union of their output
// (SPEC-007 FR-073) — e.g. holdings-derived tickers ∪ the optional MARKETDATA_WATCHLIST seed.
// Each underlying source degrades independently: a failing source is logged and skipped, never
// aborting the others, so a holdings-read failure still yields the watchlist seed and
// vice-versa (BR-074). Only a total failure (every source errors) surfaces as an error, mirroring
// the worker's own "log, don't abort" posture.
type unionSource struct {
	sources []marketdata.TickerSource
	logger  *slog.Logger
}

var _ marketdata.TickerSource = unionSource{}

// newUnionSource returns a unionSource over sources (order irrelevant — output is sorted).
func newUnionSource(logger *slog.Logger, sources ...marketdata.TickerSource) unionSource {
	return unionSource{sources: sources, logger: logger}
}

// Tickers calls every underlying source, dedupes the combined result by ticker, and returns it
// in a deterministic (alphabetical) order.
func (u unionSource) Tickers(ctx context.Context) ([]marketdata.Ticker, error) {
	seen := make(map[string]marketdata.Ticker)
	var lastErr error
	okCount := 0
	for _, s := range u.sources {
		tickers, err := s.Tickers(ctx)
		if err != nil {
			u.logger.Warn("market data: a ticker source failed; continuing with the others",
				slog.String("error", err.Error()))
			lastErr = err
			continue
		}
		okCount++
		for _, t := range tickers {
			seen[t.String()] = t
		}
	}
	if okCount == 0 && len(u.sources) > 0 {
		return nil, fmt.Errorf("union ticker source: all sources failed: %w", lastErr)
	}

	out := make([]marketdata.Ticker, 0, len(seen))
	for _, t := range seen {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out, nil
}

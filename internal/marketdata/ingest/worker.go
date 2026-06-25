// Package ingest is the market-data ingestion edge: the worker that fetches and stores
// FII quotes + macro indicators, the scheduler that runs it, and the factory that wires
// them from config (SPEC-006 FR-604/FR-605/FR-609). Observability (OTel) lives here, at
// the edge — the marketdata core stays free of OTel (BR-601, SPEC-004 conventions).
package ingest

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/platform/clock"
	"github.com/biel-ferreira/yield-forge/internal/platform/observability"
)

// Worker runs one ingestion pass: it pulls the FII watchlist and the fixed macro set from
// the provider and upserts each into its repository. Per-item failures are isolated — a bad
// ticker or a provider outage never aborts the run, and a failed fetch never overwrites
// last-known-good data (SPEC-006 BR-602, FR-605, FR-610).
type Worker struct {
	provider  marketdata.MarketDataProvider
	fiiRepo   marketdata.FIIQuoteRepository
	macroRepo marketdata.MacroRepository
	tickers   marketdata.TickerSource
	clock     clock.Clock
	logger    *slog.Logger

	tracer trace.Tracer
	runs   metric.Int64Counter
	items  metric.Int64Counter

	mu      sync.Mutex
	lastRun time.Time
}

// Summary reports the per-kind outcome of one run.
type Summary struct {
	FIIOK, FIIFailed     int
	MacroOK, MacroFailed int
}

// outcome is a low-cardinality run label for telemetry (SPEC-006 FR-609).
func (s Summary) outcome() string {
	switch {
	case s.FIIFailed+s.MacroFailed == 0:
		return "success"
	case s.FIIOK+s.MacroOK == 0:
		return "error"
	default:
		return "partial"
	}
}

func newWorker(
	provider marketdata.MarketDataProvider,
	fiiRepo marketdata.FIIQuoteRepository,
	macroRepo marketdata.MacroRepository,
	tickers marketdata.TickerSource,
	clk clock.Clock,
	logger *slog.Logger,
) *Worker {
	meter := observability.Meter("marketdata")
	runs, _ := meter.Int64Counter("marketdata.ingestion_runs")
	items, _ := meter.Int64Counter("marketdata.ingestion_items")

	w := &Worker{
		provider:  provider,
		fiiRepo:   fiiRepo,
		macroRepo: macroRepo,
		tickers:   tickers,
		clock:     clk,
		logger:    logger,
		tracer:    observability.Tracer("marketdata"),
		runs:      runs,
		items:     items,
	}

	// Freshness signal: seconds since the last completed run, so monitoring can alert if
	// ingestion stalls (SPEC-006 FR-609). No PII — it is a single duration.
	if gauge, err := meter.Int64ObservableGauge("marketdata.seconds_since_last_run"); err == nil {
		_, _ = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
			w.mu.Lock()
			last := w.lastRun
			w.mu.Unlock()
			if !last.IsZero() {
				o.ObserveInt64(gauge, int64(w.clock.Now().Sub(last)/time.Second))
			}
			return nil
		}, gauge)
	}
	return w
}

// RunOnce performs a single ingestion pass and returns its Summary. It never returns an
// error: failures are isolated, logged, metered, and degrade to keeping last-known-good.
func (w *Worker) RunOnce(ctx context.Context) Summary {
	ctx, span := w.tracer.Start(ctx, "marketdata.ingest")
	defer span.End()

	var sum Summary
	w.ingestFII(ctx, &sum)
	w.ingestMacro(ctx, &sum)

	outcome := sum.outcome()
	span.SetAttributes(
		attribute.Int("marketdata.fii_ok", sum.FIIOK),
		attribute.Int("marketdata.fii_failed", sum.FIIFailed),
		attribute.Int("marketdata.macro_ok", sum.MacroOK),
		attribute.Int("marketdata.macro_failed", sum.MacroFailed),
		attribute.String("marketdata.outcome", outcome),
	)
	if w.runs != nil {
		w.runs.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", outcome)))
	}

	w.mu.Lock()
	w.lastRun = w.clock.Now()
	w.mu.Unlock()

	w.logger.Info("market data ingestion run",
		slog.Int("fii_ok", sum.FIIOK), slog.Int("fii_failed", sum.FIIFailed),
		slog.Int("macro_ok", sum.MacroOK), slog.Int("macro_failed", sum.MacroFailed),
		slog.String("outcome", outcome))
	return sum
}

func (w *Worker) ingestFII(ctx context.Context, sum *Summary) {
	tickers, err := w.tickers.Tickers(ctx)
	if err != nil {
		w.logger.Error("market data: ticker source failed", slog.String("error", err.Error()))
		w.recordItem(ctx, "fii", "source_error")
		sum.FIIFailed++
		return
	}
	if len(tickers) == 0 {
		return // macro-only run
	}

	quotes, err := w.provider.FetchFIIQuotes(ctx, tickers)
	if err != nil {
		// Degrade: keep last-known-good for every ticker (BR-602).
		w.logger.Warn("market data: FII provider unavailable; keeping last-known-good",
			slog.String("error", err.Error()))
		w.recordItem(ctx, "fii", "provider_error")
		sum.FIIFailed += len(tickers)
		return
	}

	for t, q := range quotes {
		if err := w.fiiRepo.UpsertFIIQuote(ctx, q); err != nil {
			w.logger.Error("market data: upsert FII quote failed",
				slog.String("ticker", t.String()), slog.String("error", err.Error()))
			w.recordItem(ctx, "fii", "store_error")
			sum.FIIFailed++
			continue
		}
		w.recordItem(ctx, "fii", "success")
		sum.FIIOK++
	}
}

func (w *Worker) ingestMacro(ctx context.Context, sum *Summary) {
	for _, ind := range marketdata.AllIndicators {
		m, err := w.provider.FetchMacroIndicator(ctx, ind)
		if err != nil {
			// IFIX has no free source yet (SPEC-006 §15) and any source can be down: degrade.
			w.logger.Warn("market data: macro indicator unavailable; keeping last-known-good",
				slog.String("indicator", string(ind)), slog.String("error", err.Error()))
			w.recordItem(ctx, "macro", "provider_error")
			sum.MacroFailed++
			continue
		}
		if err := w.macroRepo.UpsertMacroIndicator(ctx, m); err != nil {
			w.logger.Error("market data: upsert macro indicator failed",
				slog.String("indicator", string(ind)), slog.String("error", err.Error()))
			w.recordItem(ctx, "macro", "store_error")
			sum.MacroFailed++
			continue
		}
		w.recordItem(ctx, "macro", "success")
		sum.MacroOK++
	}
}

func (w *Worker) recordItem(ctx context.Context, kind, outcome string) {
	if w.items != nil {
		w.items.Add(ctx, 1, metric.WithAttributes(
			attribute.String("kind", kind),
			attribute.String("outcome", outcome),
		))
	}
}

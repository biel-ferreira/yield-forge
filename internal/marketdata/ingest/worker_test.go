package ingest

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	noopTrace "go.opentelemetry.io/otel/trace/noop"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
)

// --- in-memory fakes (hand-written, no mocking lib) ---

type memFIIRepo struct {
	data map[string]marketdata.FIIQuote
	fail bool
}

func newMemFIIRepo() *memFIIRepo { return &memFIIRepo{data: map[string]marketdata.FIIQuote{}} }

func (r *memFIIRepo) UpsertFIIQuote(_ context.Context, q marketdata.FIIQuote) error {
	if r.fail {
		return io.ErrClosedPipe
	}
	r.data[q.Ticker.String()] = q
	return nil
}

func (r *memFIIRepo) GetFIIQuoteByTicker(_ context.Context, t marketdata.Ticker) (marketdata.FIIQuote, error) {
	q, ok := r.data[t.String()]
	if !ok {
		return marketdata.FIIQuote{}, marketdata.ErrFIIQuoteNotFound
	}
	return q, nil
}

type memMacroRepo struct {
	data map[marketdata.Indicator]marketdata.MacroIndicator
}

func newMemMacroRepo() *memMacroRepo {
	return &memMacroRepo{data: map[marketdata.Indicator]marketdata.MacroIndicator{}}
}

func (r *memMacroRepo) UpsertMacroIndicator(_ context.Context, m marketdata.MacroIndicator) error {
	r.data[m.Indicator] = m
	return nil
}

func (r *memMacroRepo) GetLatestMacroIndicator(_ context.Context, ind marketdata.Indicator) (marketdata.MacroIndicator, error) {
	m, ok := r.data[ind]
	if !ok {
		return marketdata.MacroIndicator{}, marketdata.ErrMacroNotFound
	}
	return m, nil
}

// failingFIIProvider errors on FetchFIIQuotes but serves macro from the embedded Fake.
type failingFIIProvider struct{ marketdata.Fake }

func (failingFIIProvider) FetchFIIQuotes(context.Context, []marketdata.Ticker) (map[marketdata.Ticker]marketdata.FIIQuote, error) {
	return nil, marketdata.ErrProviderUnavailable
}

type fakeClock struct{ t time.Time }

func (c fakeClock) Now() time.Time { return c.t }

func testWorker(t *testing.T, provider marketdata.MarketDataProvider, fii marketdata.FIIQuoteRepository, macro marketdata.MacroRepository, watch ...string) *Worker {
	t.Helper()
	wl, err := marketdata.NewWatchlist(watch)
	require.NoError(t, err)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return newWorker(provider, "fake", fii, macro, wl, fakeClock{t: time.Unix(1_700_000_000, 0).UTC()}, logger)
}

// --- span recorder helpers (in-memory OTel exporter) ---

func spanRecorder(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(noopTrace.NewTracerProvider())
	})
	return exp
}

func attr(span tracetest.SpanStub, key string) (string, bool) {
	for _, kv := range span.Attributes {
		if string(kv.Key) == key {
			return kv.Value.Emit(), true
		}
	}
	return "", false
}

// TestRunOnce_SpanCarriesMetadataOnly verifies the ingestion span records low-cardinality
// run metadata (provider + outcome + per-kind counts) and nothing sensitive (FR-609, BR-608).
func TestRunOnce_SpanCarriesMetadataOnly(t *testing.T) {
	exp := spanRecorder(t)
	w := testWorker(t, marketdata.Fake{}, newMemFIIRepo(), newMemMacroRepo(), "HGLG11")

	w.RunOnce(context.Background())

	spans := exp.GetSpans()
	require.NotEmpty(t, spans)

	var parent tracetest.SpanStub
	var sawFetchFII, sawFetchMacro bool
	for _, s := range spans {
		switch s.Name {
		case "marketdata.ingest":
			parent = s
		case "marketdata.fetch_fii":
			sawFetchFII = true
		case "marketdata.fetch_macro":
			sawFetchMacro = true
		}
	}
	require.Equal(t, "marketdata.ingest", parent.Name, "the span is named for the operation, not an id")
	require.True(t, sawFetchFII, "a child span per provider call (FR-609)")
	require.True(t, sawFetchMacro, "a child span per macro provider call (FR-609)")

	outcome, ok := attr(parent, "marketdata.outcome")
	require.True(t, ok)
	require.Equal(t, "success", outcome)

	// Only the documented low-cardinality keys appear on the parent span — no token/URL/payload.
	allowed := map[string]bool{
		"marketdata.provider": true, "marketdata.outcome": true,
		"marketdata.fii_ok": true, "marketdata.fii_failed": true,
		"marketdata.macro_ok": true, "marketdata.macro_failed": true,
	}
	for _, kv := range parent.Attributes {
		require.True(t, allowed[string(kv.Key)], "unexpected span attribute %q (possible leak)", kv.Key)
	}
}

func TestRunOnce_HappyPath(t *testing.T) {
	fiiRepo, macroRepo := newMemFIIRepo(), newMemMacroRepo()
	w := testWorker(t, marketdata.Fake{}, fiiRepo, macroRepo, "HGLG11", "KNRI11")

	sum := w.RunOnce(context.Background())

	require.Equal(t, 2, sum.FIIOK)
	require.Equal(t, 0, sum.FIIFailed)
	require.Equal(t, len(marketdata.AllIndicators), sum.MacroOK, "Fake serves every indicator incl. IFIX")
	require.Len(t, fiiRepo.data, 2)
	got, err := fiiRepo.GetFIIQuoteByTicker(context.Background(), marketdata.MustParseTicker("HGLG11"))
	require.NoError(t, err)
	require.Positive(t, got.PriceCentavos)
}

func TestRunOnce_ProviderDownKeepsLastKnownGood(t *testing.T) {
	fiiRepo, macroRepo := newMemFIIRepo(), newMemMacroRepo()
	// Seed a prior good quote; a provider outage must not erase it.
	prior := marketdata.FIIQuote{Ticker: marketdata.MustParseTicker("HGLG11"), PriceCentavos: 9_999}
	fiiRepo.data["HGLG11"] = prior

	w := testWorker(t, failingFIIProvider{}, fiiRepo, macroRepo, "HGLG11")
	sum := w.RunOnce(context.Background())

	require.Equal(t, 0, sum.FIIOK)
	require.Equal(t, 1, sum.FIIFailed)
	got, err := fiiRepo.GetFIIQuoteByTicker(context.Background(), marketdata.MustParseTicker("HGLG11"))
	require.NoError(t, err)
	require.Equal(t, int64(9_999), got.PriceCentavos, "last-known-good is preserved (BR-602)")
	require.Equal(t, "partial", sum.outcome(), "macro still succeeded, FII degraded")
}

func TestRunOnce_StoreFailureIsIsolated(t *testing.T) {
	fiiRepo := newMemFIIRepo()
	fiiRepo.fail = true // every FII upsert fails
	macroRepo := newMemMacroRepo()

	w := testWorker(t, marketdata.Fake{}, fiiRepo, macroRepo, "HGLG11")
	sum := w.RunOnce(context.Background())

	require.Equal(t, 1, sum.FIIFailed)
	require.Equal(t, len(marketdata.AllIndicators), sum.MacroOK, "a FII store error doesn't stop macro ingestion")
}

// TestRunOnce_TickerSourceFailureKeepsLastKnownGood proves the SPEC-007 composition (a
// unionSource over holdings + watchlist) degrades through the worker's existing source_error
// path (worker.go:141-147) exactly like the old plain-Watchlist source did — a total ticker
// source failure never erases last-known-good fii_quotes and macro ingestion still runs.
func TestRunOnce_TickerSourceFailureKeepsLastKnownGood(t *testing.T) {
	fiiRepo, macroRepo := newMemFIIRepo(), newMemMacroRepo()
	prior := marketdata.FIIQuote{Ticker: marketdata.MustParseTicker("HGLG11"), PriceCentavos: 9_999}
	fiiRepo.data["HGLG11"] = prior

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	boom := fakeSource{err: errors.New("holdings read failed")}
	tickers := newUnionSource(logger, boom) // every source fails -> union itself errors

	w := newWorker(marketdata.Fake{}, "fake", fiiRepo, macroRepo, tickers,
		fakeClock{t: time.Unix(1_700_000_000, 0).UTC()}, logger)
	sum := w.RunOnce(context.Background())

	require.Equal(t, 0, sum.FIIOK)
	require.Equal(t, 1, sum.FIIFailed)
	require.Equal(t, len(marketdata.AllIndicators), sum.MacroOK, "macro still ingests despite the ticker source failure")
	got, err := fiiRepo.GetFIIQuoteByTicker(context.Background(), marketdata.MustParseTicker("HGLG11"))
	require.NoError(t, err)
	require.Equal(t, int64(9_999), got.PriceCentavos, "last-known-good is preserved (BR-602/BR-074)")
}

func TestRunOnce_EmptyWatchlistIngestsMacroOnly(t *testing.T) {
	fiiRepo, macroRepo := newMemFIIRepo(), newMemMacroRepo()
	w := testWorker(t, marketdata.Fake{}, fiiRepo, macroRepo) // no tickers

	sum := w.RunOnce(context.Background())
	require.Equal(t, 0, sum.FIIOK)
	require.Equal(t, 0, sum.FIIFailed)
	require.Equal(t, len(marketdata.AllIndicators), sum.MacroOK)
	require.Empty(t, fiiRepo.data)
}

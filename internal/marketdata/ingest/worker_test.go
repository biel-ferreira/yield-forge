package ingest

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

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
	return newWorker(provider, fii, macro, wl, fakeClock{t: time.Unix(1_700_000_000, 0).UTC()}, logger)
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

func TestRunOnce_EmptyWatchlistIngestsMacroOnly(t *testing.T) {
	fiiRepo, macroRepo := newMemFIIRepo(), newMemMacroRepo()
	w := testWorker(t, marketdata.Fake{}, fiiRepo, macroRepo) // no tickers

	sum := w.RunOnce(context.Background())
	require.Equal(t, 0, sum.FIIOK)
	require.Equal(t, 0, sum.FIIFailed)
	require.Equal(t, len(marketdata.AllIndicators), sum.MacroOK)
	require.Empty(t, fiiRepo.data)
}

package fii_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/marketdata/fii"
)

// fakeFundamentals returns a fixed quote (no last-dividend) per requested ticker, or a
// preset error to exercise the degrade path.
type fakeFundamentals struct{ err error }

func (f fakeFundamentals) FetchFIIQuotes(_ context.Context, tickers []marketdata.Ticker) (map[marketdata.Ticker]marketdata.FIIQuote, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make(map[marketdata.Ticker]marketdata.FIIQuote, len(tickers))
	for _, t := range tickers {
		out[t] = marketdata.FIIQuote{Ticker: t, PriceCentavos: 10_000, Sector: marketdata.SectorLogistics}
	}
	return out, nil
}

// fakeDividends returns per-ticker results: a cents value, or an error for "fail" tickers.
type fakeDividends struct {
	cents map[string]int64
	fail  map[string]bool
}

func (f fakeDividends) FetchLastDividend(_ context.Context, t marketdata.Ticker) (int64, *time.Time, error) {
	if f.fail[t.String()] {
		return 0, nil, marketdata.ErrProviderUnavailable
	}
	c := f.cents[t.String()]
	if c == 0 {
		return 0, nil, nil
	}
	d := time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC)
	return c, &d, nil
}

func TestComposite_EnrichesLastDividend(t *testing.T) {
	hglg := marketdata.MustParseTicker("HGLG11")
	knri := marketdata.MustParseTicker("KNRI11")
	c := fii.New(
		fakeFundamentals{},
		fakeDividends{cents: map[string]int64{"HGLG11": 110}}, // KNRI11 has none
	)

	got, err := c.FetchFIIQuotes(context.Background(), []marketdata.Ticker{hglg, knri})
	require.NoError(t, err)

	require.Equal(t, int64(110), got[hglg].LastDividendCentavos)
	require.NotNil(t, got[hglg].LastDividendDate)
	require.Zero(t, got[knri].LastDividendCentavos, "no dividend -> left empty (partial quote)")
	require.Nil(t, got[knri].LastDividendDate)
}

func TestComposite_DividendFailureIsBestEffort(t *testing.T) {
	hglg := marketdata.MustParseTicker("HGLG11")
	c := fii.New(
		fakeFundamentals{},
		fakeDividends{fail: map[string]bool{"HGLG11": true}},
	)

	got, err := c.FetchFIIQuotes(context.Background(), []marketdata.Ticker{hglg})
	require.NoError(t, err, "a dividend lookup failure must not fail the quote")
	require.Equal(t, int64(10_000), got[hglg].PriceCentavos, "fundamentals are still returned")
	require.Zero(t, got[hglg].LastDividendCentavos)
}

func TestComposite_FundamentalsFailureDegrades(t *testing.T) {
	boom := errors.New("boom")
	c := fii.New(fakeFundamentals{err: boom}, fakeDividends{})

	_, err := c.FetchFIIQuotes(context.Background(), []marketdata.Ticker{marketdata.MustParseTicker("HGLG11")})
	require.ErrorIs(t, err, boom, "fundamentals is a hard dependency — its failure degrades the batch")
}

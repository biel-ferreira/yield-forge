package ingest

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
)

// fakeSource is a hand-written fake marketdata.TickerSource (no mocking lib).
type fakeSource struct {
	tickers []marketdata.Ticker
	err     error
}

func (f fakeSource) Tickers(context.Context) ([]marketdata.Ticker, error) { return f.tickers, f.err }

func tks(ss ...string) []marketdata.Ticker {
	out := make([]marketdata.Ticker, len(ss))
	for i, s := range ss {
		out[i] = marketdata.MustParseTicker(s)
	}
	return out
}

func TestUnionSource_DedupesAndSorts(t *testing.T) {
	holdings := fakeSource{tickers: tks("HGLG11", "MXRF11")}
	watchlist := fakeSource{tickers: tks("MXRF11", "XPLG11")} // MXRF11 overlaps

	u := newUnionSource(testLogger(), holdings, watchlist)
	got, err := u.Tickers(context.Background())
	require.NoError(t, err)

	gotStr := make([]string, len(got))
	for i, tk := range got {
		gotStr[i] = tk.String()
	}
	require.Equal(t, []string{"HGLG11", "MXRF11", "XPLG11"}, gotStr, "deduped, alphabetically ordered")
}

func TestUnionSource_DegradesPerSource(t *testing.T) {
	failing := fakeSource{err: errors.New("holdings read failed")}
	watchlist := fakeSource{tickers: tks("HGLG11")}

	u := newUnionSource(testLogger(), failing, watchlist)
	got, err := u.Tickers(context.Background())
	require.NoError(t, err, "one source failing must not fail the union (BR-074)")
	require.Len(t, got, 1)
	require.Equal(t, "HGLG11", got[0].String(), "the watchlist seed still comes through")
}

func TestUnionSource_AllSourcesFail(t *testing.T) {
	boom := errors.New("boom")
	u := newUnionSource(testLogger(), fakeSource{err: boom}, fakeSource{err: boom})

	_, err := u.Tickers(context.Background())
	require.Error(t, err, "only a total failure surfaces as an error")
}

func TestUnionSource_NoSourcesIsEmptyNotError(t *testing.T) {
	u := newUnionSource(testLogger())
	got, err := u.Tickers(context.Background())
	require.NoError(t, err)
	require.Empty(t, got)
}

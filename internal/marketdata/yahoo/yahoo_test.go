package yahoo_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/marketdata/yahoo"
)

func newServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.True(t, strings.HasSuffix(r.URL.Path, "/HGLG11.SA"), "the .SA suffix is appended: %s", r.URL.Path)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

const twoDividends = `{"chart":{"result":[{"events":{"dividends":{
	"1715000000":{"amount":1.05,"date":1715000000},
	"1717200000":{"amount":1.10,"date":1717200000}
}}}]}}`

func TestFetchLastDividend_PicksMostRecent(t *testing.T) {
	srv := newServer(t, http.StatusOK, twoDividends)
	a := yahoo.New(srv.URL, 5*time.Second)

	cents, date, err := a.FetchLastDividend(context.Background(), marketdata.MustParseTicker("HGLG11"))
	require.NoError(t, err)
	require.Equal(t, int64(110), cents, "the later distribution (1,10) wins")
	require.NotNil(t, date)
	want := time.Unix(1717200000, 0).UTC()
	require.Equal(t, want.Year(), date.Year())
	require.Equal(t, want.YearDay(), date.YearDay())
	require.Equal(t, 0, date.Hour(), "date is truncated to the day")
}

func TestFetchLastDividend_NoDividends(t *testing.T) {
	srv := newServer(t, http.StatusOK, `{"chart":{"result":[{"events":{}}]}}`)
	a := yahoo.New(srv.URL, 5*time.Second)

	cents, date, err := a.FetchLastDividend(context.Background(), marketdata.MustParseTicker("HGLG11"))
	require.NoError(t, err, "no dividends is a legitimate empty result, not an error")
	require.Zero(t, cents)
	require.Nil(t, date)
}

func TestFetchLastDividend_HTTPErrorDegrades(t *testing.T) {
	srv := newServer(t, http.StatusTooManyRequests, "rate limited")
	a := yahoo.New(srv.URL, 5*time.Second)

	_, _, err := a.FetchLastDividend(context.Background(), marketdata.MustParseTicker("HGLG11"))
	require.ErrorIs(t, err, marketdata.ErrProviderUnavailable)
}

func TestFetchLastDividend_MalformedDegrades(t *testing.T) {
	srv := newServer(t, http.StatusOK, "not json")
	a := yahoo.New(srv.URL, 5*time.Second)

	_, _, err := a.FetchLastDividend(context.Background(), marketdata.MustParseTicker("HGLG11"))
	require.ErrorIs(t, err, marketdata.ErrProviderUnavailable)
}

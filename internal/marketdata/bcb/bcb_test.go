package bcb_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/marketdata/bcb"
)

type fakeClock struct{ t time.Time }

func (c fakeClock) Now() time.Time { return c.t }

func newServer(t *testing.T, wantCode int, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if wantCode > 0 {
			require.Contains(t, r.URL.Path, fmt.Sprintf("bcdata.sgs.%d", wantCode), "series code in path")
		}
		require.Equal(t, "json", r.URL.Query().Get("formato"))
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFetchMacroIndicator_Selic(t *testing.T) {
	srv := newServer(t, 432, http.StatusOK, `[{"data":"01/06/2026","valor":"10.50"}]`)
	now := time.Date(2026, 6, 21, 9, 0, 0, 0, time.UTC)
	a := bcb.New(srv.URL, 5*time.Second, fakeClock{t: now})

	got, err := a.FetchMacroIndicator(context.Background(), marketdata.IndicatorSELIC)
	require.NoError(t, err)
	require.Equal(t, marketdata.IndicatorSELIC, got.Indicator)
	require.Equal(t, int64(1050), got.Value, "10.50% -> 1050 bps")
	require.Equal(t, marketdata.UnitBps, got.Unit)
	require.Equal(t, "bcb-sgs", got.Source)
	require.Equal(t, now, got.FetchedAt)
	require.Equal(t, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), got.ReferenceDate)
}

func TestFetchMacroIndicator_IPCASeriesCode(t *testing.T) {
	srv := newServer(t, 13522, http.StatusOK, `[{"data":"31/05/2026","valor":"3.93"}]`)
	a := bcb.New(srv.URL, 5*time.Second, fakeClock{t: time.Unix(0, 0).UTC()})

	got, err := a.FetchMacroIndicator(context.Background(), marketdata.IndicatorIPCA)
	require.NoError(t, err)
	require.Equal(t, int64(393), got.Value, "3.93% -> 393 bps")
}

func TestFetchMacroIndicator_IFIXUnsupported(t *testing.T) {
	// No HTTP server should be hit — IFIX has no BCB series, so it fails before any request.
	a := bcb.New("http://127.0.0.1:0", 5*time.Second, fakeClock{t: time.Unix(0, 0).UTC()})

	_, err := a.FetchMacroIndicator(context.Background(), marketdata.IndicatorIFIX)
	require.ErrorIs(t, err, marketdata.ErrProviderUnavailable)
	require.Contains(t, err.Error(), "ifix")
}

func TestFetchMacroIndicator_Degrades(t *testing.T) {
	cases := map[string]struct {
		status int
		body   string
	}{
		"http error":     {http.StatusInternalServerError, "boom"},
		"empty series":   {http.StatusOK, `[]`},
		"malformed json": {http.StatusOK, "not json"},
		"bad value":      {http.StatusOK, `[{"data":"01/06/2026","valor":"x"}]`},
		"bad date":       {http.StatusOK, `[{"data":"2026-06-01","valor":"10.50"}]`},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			srv := newServer(t, 432, c.status, c.body)
			a := bcb.New(srv.URL, 5*time.Second, fakeClock{t: time.Unix(0, 0).UTC()})
			_, err := a.FetchMacroIndicator(context.Background(), marketdata.IndicatorSELIC)
			require.ErrorIs(t, err, marketdata.ErrProviderUnavailable)
		})
	}
}

func TestFetchMacroIndicator_LiveBCB_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live BCB test in -short mode")
	}
	if strings.TrimSpace(os.Getenv("TEST_BCB_LIVE")) == "" {
		t.Skip("set TEST_BCB_LIVE=1 to hit the real BCB SGS API")
	}
	a := bcb.New("https://api.bcb.gov.br/dados/serie", 15*time.Second, fakeClock{t: time.Now().UTC()})
	got, err := a.FetchMacroIndicator(context.Background(), marketdata.IndicatorSELIC)
	require.NoError(t, err)
	require.Positive(t, got.Value, "a live Selic should be positive")
}

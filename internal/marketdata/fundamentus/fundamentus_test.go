package fundamentus_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/marketdata/fundamentus"
)

type fakeClock struct{ t time.Time }

func (c fakeClock) Now() time.Time { return c.t }

// tableHTML mimics the Fundamentus fii_resultado.php table (header + rows). Columns are
// deliberately not in the order the adapter reads them, to prove header-keyed parsing.
const tableHTML = `<!doctype html><html><body>
<table id="resultado">
<thead><tr>
  <th>Papel</th><th>Segmento</th><th>Cotação</th><th>FFO Yield</th>
  <th>Dividend Yield</th><th>P/VP</th><th>Valor de Mercado</th>
</tr></thead>
<tbody>
  <tr><td>HGLG11</td><td>Logística</td><td>157,75</td><td>7,00%</td><td>8,50%</td><td>0,95</td><td>1.000.000</td></tr>
  <tr><td>KNRI11</td><td>Híbrido</td><td>148,20</td><td>6,50%</td><td>7,80%</td><td>0,88</td><td>900.000</td></tr>
  <tr><td>MXRF11</td><td>Papel</td><td>10,30</td><td>11,0%</td><td>12,40%</td><td>1,01</td><td>800.000</td></tr>
</tbody></table></body></html>`

func newServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/fii_resultado.php", r.URL.Path)
		require.NotEmpty(t, r.Header.Get("User-Agent"), "a descriptive User-Agent is sent")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFetchFIIQuotes_Success(t *testing.T) {
	srv := newServer(t, http.StatusOK, tableHTML)
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	a := fundamentus.New(srv.URL, 5*time.Second, fakeClock{t: now})

	hglg := marketdata.MustParseTicker("HGLG11")
	knri := marketdata.MustParseTicker("KNRI11")
	got, err := a.FetchFIIQuotes(context.Background(), []marketdata.Ticker{hglg, knri})
	require.NoError(t, err)
	require.Len(t, got, 2, "only the requested tickers are returned (MXRF11 omitted)")

	q := got[hglg]
	require.Equal(t, int64(15_775), q.PriceCentavos)
	require.Equal(t, 850, q.DividendYieldBps)
	require.Equal(t, 9_500, q.PVPBps)
	require.Equal(t, marketdata.SectorLogistics, q.Sector)
	require.Equal(t, "fundamentus", q.Source)
	require.Equal(t, now, q.FetchedAt)
	require.Nil(t, q.LastDividendDate, "last-dividend is enriched later by Yahoo, not here")

	require.Equal(t, marketdata.SectorHybrid, got[knri].Sector)
	require.Equal(t, int64(14_820), got[knri].PriceCentavos)
}

func TestFetchFIIQuotes_UnknownTickerAbsentNotError(t *testing.T) {
	srv := newServer(t, http.StatusOK, tableHTML)
	a := fundamentus.New(srv.URL, 5*time.Second, fakeClock{t: time.Unix(0, 0).UTC()})

	got, err := a.FetchFIIQuotes(context.Background(), []marketdata.Ticker{
		marketdata.MustParseTicker("HGLG11"),
		marketdata.MustParseTicker("XPLG11"), // not in the table
	})
	require.NoError(t, err)
	require.Len(t, got, 1)
	_, ok := got[marketdata.MustParseTicker("XPLG11")]
	require.False(t, ok, "a ticker missing from the table is absent, not an error")
}

func TestFetchFIIQuotes_LayoutChangeDegrades(t *testing.T) {
	// A table whose headers no longer contain the expected columns.
	const bad = `<table><tr><th>Col1</th><th>Col2</th></tr><tr><td>x</td><td>y</td></tr></table>`
	srv := newServer(t, http.StatusOK, bad)
	a := fundamentus.New(srv.URL, 5*time.Second, fakeClock{t: time.Unix(0, 0).UTC()})

	_, err := a.FetchFIIQuotes(context.Background(), []marketdata.Ticker{marketdata.MustParseTicker("HGLG11")})
	require.ErrorIs(t, err, marketdata.ErrProviderUnavailable)
}

func TestFetchFIIQuotes_HTTPErrorDegrades(t *testing.T) {
	srv := newServer(t, http.StatusInternalServerError, "boom")
	a := fundamentus.New(srv.URL, 5*time.Second, fakeClock{t: time.Unix(0, 0).UTC()})

	_, err := a.FetchFIIQuotes(context.Background(), []marketdata.Ticker{marketdata.MustParseTicker("HGLG11")})
	require.ErrorIs(t, err, marketdata.ErrProviderUnavailable)
}

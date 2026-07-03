package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/auth"
	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/platform/buildinfo"
)

// fakeMarketDataReader returns a configured value per indicator, or an error for any indicator
// not present in vals (simulating "not yet ingested" — SPEC-006 ErrMacroNotFound).
type fakeMarketDataReader struct {
	vals map[marketdata.Indicator]marketdata.MacroIndicator
}

func (f fakeMarketDataReader) GetLatestMacroIndicator(_ context.Context, ind marketdata.Indicator) (marketdata.MacroIndicator, error) {
	if m, ok := f.vals[ind]; ok {
		return m, nil
	}
	return marketdata.MacroIndicator{}, marketdata.ErrMacroNotFound
}

func newMarketHandler(r MarketDataReader) marketHandler {
	return marketHandler{reader: r, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

// marketRouter builds a full router (auth middleware + otelhttp) for the span/route tests.
func marketRouter(r MarketDataReader) http.Handler {
	user := auth.User{ID: "u1", Email: "me@example.com"}
	return NewRouter(Deps{
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Build:      buildinfo.Info{},
		Ready:      fakePinger{},
		Auth:       fakeAuth{authUser: user},
		Markets:    r,
		CookieName: "yf_session",
		SessionTTL: time.Hour,
	})
}

func TestGetMarketIndicators_ReturnsAllThreeWhenPresent(t *testing.T) {
	refDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	reader := fakeMarketDataReader{vals: map[marketdata.Indicator]marketdata.MacroIndicator{
		marketdata.IndicatorSELIC: {Indicator: marketdata.IndicatorSELIC, Value: 1_075, ReferenceDate: refDate},
		marketdata.IndicatorCDI:   {Indicator: marketdata.IndicatorCDI, Value: 1_050, ReferenceDate: refDate},
		marketdata.IndicatorIPCA:  {Indicator: marketdata.IndicatorIPCA, Value: 450, ReferenceDate: refDate},
	}}
	h := newMarketHandler(reader)

	rec := httptest.NewRecorder()
	h.getMarketIndicators(rec, authed(http.MethodGet, "/market/indicators", "", "u1"))

	require.Equal(t, http.StatusOK, rec.Code)
	var out []marketIndicatorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Len(t, out, 3)
	require.Equal(t, "2026-07-01", out[0].ReferenceDate)
	require.Equal(t, int64(1_075), out[0].ValueBps)
}

func TestGetMarketIndicators_DegradesGracefullyWhenOneMissing(t *testing.T) {
	refDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	// Only CDI ingested — SELIC/IPCA absent (fresh environment, BR-1094).
	reader := fakeMarketDataReader{vals: map[marketdata.Indicator]marketdata.MacroIndicator{
		marketdata.IndicatorCDI: {Indicator: marketdata.IndicatorCDI, Value: 1_050, ReferenceDate: refDate},
	}}
	h := newMarketHandler(reader)

	rec := httptest.NewRecorder()
	h.getMarketIndicators(rec, authed(http.MethodGet, "/market/indicators", "", "u1"))

	require.Equal(t, http.StatusOK, rec.Code, "a partial/missing indicator never fails the whole request")
	var out []marketIndicatorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Len(t, out, 1)
	require.Equal(t, "cdi", out[0].Indicator)
}

func TestGetMarketIndicators_NoneIngestedYieldsEmptyList(t *testing.T) {
	h := newMarketHandler(fakeMarketDataReader{vals: map[marketdata.Indicator]marketdata.MacroIndicator{}})

	rec := httptest.NewRecorder()
	h.getMarketIndicators(rec, authed(http.MethodGet, "/market/indicators", "", "u1"))

	require.Equal(t, http.StatusOK, rec.Code, "never a 500 — an empty environment is not an error")
	var out []marketIndicatorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Empty(t, out)
}

func TestGetMarketIndicators_Unauthenticated(t *testing.T) {
	h := newMarketHandler(fakeMarketDataReader{})
	rec := httptest.NewRecorder()
	h.getMarketIndicators(rec, httptest.NewRequest(http.MethodGet, "/market/indicators", nil))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestHTTP_MarketIndicatorsSpanRouteNamed verifies the new endpoint's span is auto-applied and
// route-named by the existing otelhttp + routeNamer middleware (SPEC-004 FR-403) — "verify,
// don't assume" (SPEC-109 Phase 5). Indicator values are NOT secret (public economic data,
// SPEC-109 §10), so — unlike the holdings span test — this does not assert their absence.
func TestHTTP_MarketIndicatorsSpanRouteNamed(t *testing.T) {
	exp := spanRecorder(t)
	refDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	router := marketRouter(fakeMarketDataReader{vals: map[marketdata.Indicator]marketdata.MacroIndicator{
		marketdata.IndicatorCDI: {Indicator: marketdata.IndicatorCDI, Value: 1_050, ReferenceDate: refDate},
	}})

	rr := doReq(router, http.MethodGet, "/market/indicators", "", &http.Cookie{Name: "yf_session", Value: "tok"})
	require.Equal(t, http.StatusOK, rr.Code)

	spans := exp.GetSpans()
	require.Len(t, spans, 1)
	require.Equal(t, "GET /market/indicators", spans[0].Name, "route-named, not the raw path — low cardinality")
}

package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/insight"
	"github.com/biel-ferreira/yield-forge/internal/insight/engine"
)

type fakeInsightsEngine struct {
	gotUserID string
	result    engine.Insights
	err       error
}

func (f *fakeInsightsEngine) Insights(_ context.Context, userID string) (engine.Insights, error) {
	f.gotUserID = userID
	return f.result, f.err
}

func newInsightsHandler(svc InsightsEngine) insightsHandler {
	return insightsHandler{service: svc, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func sampleInsights() engine.Insights {
	return engine.Insights{
		Items: []insight.Insight{
			{Category: "portfolio", Title: "Concentração em logística", Detail: "...", Explanation: "porque ..."},
		},
		Disclaimer: insight.Disclaimer,
		Available:  true,
	}
}

func TestGetInsights_ContextIdentityAndShape(t *testing.T) {
	svc := &fakeInsightsEngine{result: sampleInsights()}
	h := newInsightsHandler(svc)

	rec := httptest.NewRecorder()
	h.getInsights(rec, authed(http.MethodGet, "/insights", "", "u1"))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "u1", svc.gotUserID, "engine called with the context user_id")

	var resp insightsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.True(t, resp.Available)
	require.Len(t, resp.Insights, 1)
	require.Equal(t, "portfolio", resp.Insights[0].Category)
	require.NotEmpty(t, resp.Insights[0].Explanation, "every insight carries an explanation")
	require.NotEmpty(t, resp.Disclaimer, "the non-advice disclaimer is present")
}

func TestGetInsights_DegradedIsStill200(t *testing.T) {
	svc := &fakeInsightsEngine{result: engine.Insights{Disclaimer: insight.Disclaimer, Available: false}}
	h := newInsightsHandler(svc)
	rec := httptest.NewRecorder()
	h.getInsights(rec, authed(http.MethodGet, "/insights", "", "u1"))

	require.Equal(t, http.StatusOK, rec.Code, "an LLM outage is a 200 available:false, not a 5xx")
	var resp insightsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.False(t, resp.Available)
	require.NotNil(t, resp.Insights, "insights serializes as [] not null")
	require.Empty(t, resp.Insights)
}

func TestGetInsights_ServiceError(t *testing.T) {
	svc := &fakeInsightsEngine{err: errors.New("boom")}
	h := newInsightsHandler(svc)
	rec := httptest.NewRecorder()
	h.getInsights(rec, authed(http.MethodGet, "/insights", "", "u1"))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetInsights_Unauthenticated(t *testing.T) {
	h := newInsightsHandler(&fakeInsightsEngine{})
	rec := httptest.NewRecorder()
	h.getInsights(rec, httptest.NewRequest(http.MethodGet, "/insights", nil))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

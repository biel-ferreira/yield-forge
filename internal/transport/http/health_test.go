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

	"github.com/biel-ferreira/yield-forge/internal/health"
	"github.com/biel-ferreira/yield-forge/internal/insight"
)

type fakeHealthScorer struct {
	gotUserID string
	result    health.HealthScore
	err       error
}

func (f *fakeHealthScorer) Score(_ context.Context, userID string) (health.HealthScore, error) {
	f.gotUserID = userID
	return f.result, f.err
}

func newHealthHandler(svc HealthScorer) healthHandler {
	return healthHandler{service: svc, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func sampleHealth() health.HealthScore {
	return health.HealthScore{
		Score: 72,
		Factors: []health.FactorScore{
			{Factor: health.FactorDiversification, Score: 80, WeightBps: 2500, Explanation: "5 posições"},
			{Factor: health.FactorConcentration, Score: 55, WeightBps: 2500, Explanation: "45% no maior"},
		},
		Narrative: "Carteira saudável.", NarrativeAvailable: true, Disclaimer: insight.Disclaimer,
	}
}

func TestGetHealthScore_IdentityAndShape(t *testing.T) {
	svc := &fakeHealthScorer{result: sampleHealth()}
	h := newHealthHandler(svc)

	rec := httptest.NewRecorder()
	h.getHealthScore(rec, authed(http.MethodGet, "/health-score", "", "u1"))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "u1", svc.gotUserID, "identity from context")

	var resp healthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 72, resp.Score)
	require.Len(t, resp.Factors, 2)
	require.Equal(t, "diversification", resp.Factors[0].Name)
	require.Equal(t, 2500, resp.Factors[0].WeightBps)
	require.NotEmpty(t, resp.Factors[0].Explanation)
	require.True(t, resp.NarrativeAvailable)
	require.NotEmpty(t, resp.Narrative)
}

func TestGetHealthScore_NarrativeDegradedStill200(t *testing.T) {
	svc := &fakeHealthScorer{result: health.HealthScore{
		Score:   60,
		Factors: []health.FactorScore{{Factor: health.FactorLiquidity, Score: 60, WeightBps: 10000, Explanation: "x"}},
		// narrative omitted (LLM down)
	}}
	h := newHealthHandler(svc)
	rec := httptest.NewRecorder()
	h.getHealthScore(rec, authed(http.MethodGet, "/health-score", "", "u1"))

	require.Equal(t, http.StatusOK, rec.Code)
	var resp healthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 60, resp.Score, "the score is present even with no narrative")
	require.False(t, resp.NarrativeAvailable)
	require.Empty(t, resp.Narrative)
	require.NotNil(t, resp.Factors)
}

func TestGetHealthScore_ServiceError(t *testing.T) {
	h := newHealthHandler(&fakeHealthScorer{err: errors.New("boom")})
	rec := httptest.NewRecorder()
	h.getHealthScore(rec, authed(http.MethodGet, "/health-score", "", "u1"))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetHealthScore_Unauthenticated(t *testing.T) {
	h := newHealthHandler(&fakeHealthScorer{})
	rec := httptest.NewRecorder()
	h.getHealthScore(rec, httptest.NewRequest(http.MethodGet, "/health-score", nil))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

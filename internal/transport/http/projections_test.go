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

	"github.com/biel-ferreira/yield-forge/internal/projection"
)

type fakeProjectionEngine struct {
	gotUserID       string
	gotContribution int64
	gotHorizon      int
	result          projection.Projections
	err             error
}

func (f *fakeProjectionEngine) Project(_ context.Context, userID string, contribution int64, horizon int) (projection.Projections, error) {
	f.gotUserID = userID
	f.gotContribution = contribution
	f.gotHorizon = horizon
	return f.result, f.err
}

func newProjectionsHandler(svc ProjectionEngine) projectionsHandler {
	return projectionsHandler{service: svc, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func sampleProjections() projection.Projections {
	return projection.Projections{
		Income: []projection.ScenarioIncome{
			{Scenario: projection.ScenarioBase, MonthlyCentavos: 667, AnnualCentavos: 8_000},
		},
		NetWorth: []projection.ScenarioNetWorth{
			{Scenario: projection.ScenarioBase, Points: []projection.NetWorthPoint{{Year: 0, ValueCentavos: 100_000}}},
		},
		Disclaimer: projection.Disclaimer,
	}
}

func TestGetProjections_IdentityShapeAndDefaults(t *testing.T) {
	svc := &fakeProjectionEngine{result: sampleProjections()}
	h := newProjectionsHandler(svc)

	rec := httptest.NewRecorder()
	h.getProjections(rec, authed(http.MethodGet, "/projections", "", "u1"))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "u1", svc.gotUserID, "identity from context")
	require.Equal(t, int64(0), svc.gotContribution, "default contribution 0")
	require.Equal(t, 10, svc.gotHorizon, "default horizon 10")

	var resp projectionsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Income, 1)
	require.Equal(t, "base", resp.Income[0].Scenario)
	require.Len(t, resp.NetWorth, 1)
	require.NotEmpty(t, resp.Disclaimer)
}

func TestGetProjections_ParsesParams(t *testing.T) {
	svc := &fakeProjectionEngine{result: sampleProjections()}
	h := newProjectionsHandler(svc)
	rec := httptest.NewRecorder()
	h.getProjections(rec, authed(http.MethodGet, "/projections?monthly_contribution_centavos=50000&horizon_years=20", "", "u1"))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(50_000), svc.gotContribution)
	require.Equal(t, 20, svc.gotHorizon)
}

func TestGetProjections_RejectsBadParams(t *testing.T) {
	for _, query := range []string{
		"?monthly_contribution_centavos=-1",
		"?monthly_contribution_centavos=500.5", // float rejected
		"?monthly_contribution_centavos=abc",
		"?horizon_years=0",
		"?horizon_years=41",
		"?horizon_years=xyz",
	} {
		t.Run(query, func(t *testing.T) {
			svc := &fakeProjectionEngine{result: sampleProjections()}
			h := newProjectionsHandler(svc)
			rec := httptest.NewRecorder()
			h.getProjections(rec, authed(http.MethodGet, "/projections"+query, "", "u1"))
			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.Empty(t, svc.gotUserID, "no service call on a bad request")
		})
	}
}

func TestGetProjections_ServiceError(t *testing.T) {
	h := newProjectionsHandler(&fakeProjectionEngine{err: errors.New("boom")})
	rec := httptest.NewRecorder()
	h.getProjections(rec, authed(http.MethodGet, "/projections", "", "u1"))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetProjections_Unauthenticated(t *testing.T) {
	h := newProjectionsHandler(&fakeProjectionEngine{})
	rec := httptest.NewRecorder()
	h.getProjections(rec, httptest.NewRequest(http.MethodGet, "/projections", nil))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

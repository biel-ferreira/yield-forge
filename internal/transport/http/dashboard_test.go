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
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/dashboard"
	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/portfolio"
)

type fakeDashboardService struct {
	gotUserID string
	result    dashboard.Dashboard
	err       error
}

func (f *fakeDashboardService) GetDashboard(_ context.Context, userID string) (dashboard.Dashboard, error) {
	f.gotUserID = userID
	return f.result, f.err
}

func newDashboardHandler(svc DashboardService) dashboardHandler {
	return dashboardHandler{service: svc, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func sampleDashboard() dashboard.Dashboard {
	return dashboard.Dashboard{
		Summary: dashboard.Summary{
			TotalInvestedCentavos: 1_000_000, CurrentValueCentavos: 1_100_000,
			MonthlyIncomeCentavos: 5_000, GrowthCentavos: 100_000, GrowthBps: 1_000,
		},
		Allocation: []dashboard.ClassSlice{{Class: dashboard.ClassFII, ValueCentavos: 1_100_000, ShareBps: 10_000}},
		FIISectors: []dashboard.SectorSlice{{Sector: marketdata.SectorLogistics, ValueCentavos: 1_100_000, ShareBps: 10_000}},
	}
}

func TestGetDashboard_ContextIdentityAndMoney(t *testing.T) {
	svc := &fakeDashboardService{result: sampleDashboard()}
	h := newDashboardHandler(svc)

	rec := httptest.NewRecorder()
	h.getDashboard(rec, authed(http.MethodGet, "/dashboard", "", "u1"))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "u1", svc.gotUserID, "service called with the context user_id")

	var resp dashboardResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, int64(1_100_000), resp.Summary.CurrentValueCentavos, "patrimony as integer centavos")
	require.Equal(t, 1_000, resp.Summary.GrowthBps)
	require.NotNil(t, resp.StaleTickers, "stale_tickers serializes as [] not null")
}

func TestGetDashboard_Empty(t *testing.T) {
	svc := &fakeDashboardService{result: dashboard.Compute(portfolio.Holdings{}, nil, time.Unix(0, 0).UTC())}
	h := newDashboardHandler(svc)
	rec := httptest.NewRecorder()
	h.getDashboard(rec, authed(http.MethodGet, "/dashboard", "", "u1"))
	require.Equal(t, http.StatusOK, rec.Code)

	var resp dashboardResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Zero(t, resp.Summary.CurrentValueCentavos)
	require.Len(t, resp.Allocation, 4)
	require.Empty(t, resp.StaleTickers)
}

func TestGetDashboard_ServiceError(t *testing.T) {
	svc := &fakeDashboardService{err: errors.New("boom")}
	h := newDashboardHandler(svc)
	rec := httptest.NewRecorder()
	h.getDashboard(rec, authed(http.MethodGet, "/dashboard", "", "u1"))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetDashboard_Unauthenticated(t *testing.T) {
	h := newDashboardHandler(&fakeDashboardService{})
	rec := httptest.NewRecorder()
	h.getDashboard(rec, httptest.NewRequest(http.MethodGet, "/dashboard", nil))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/biel-ferreira/yield-forge/internal/dashboard"
)

// DashboardService is the slice of the dashboard service the transport needs. Consumer-defined
// here so the handler stays testable with a small fake; *dashboard.Service satisfies it (SPEC-103).
type DashboardService interface {
	GetDashboard(ctx context.Context, userID string) (dashboard.Dashboard, error)
}

// dashboardHandler serves GET /dashboard.
type dashboardHandler struct {
	service DashboardService
	logger  *slog.Logger
}

// --- DTOs (money crosses the wire as integer centavos / bps, never a float — BR-1032) ---

type dashboardResponse struct {
	Summary      summaryResponse       `json:"summary"`
	Allocation   []classSliceResponse  `json:"allocation"`
	FIISectors   []sectorSliceResponse `json:"fii_sectors"`
	StaleTickers []string              `json:"stale_tickers"`
}

type summaryResponse struct {
	TotalInvestedCentavos int64 `json:"total_invested_centavos"`
	CurrentValueCentavos  int64 `json:"current_value_centavos"` // the full patrimony / net worth
	MonthlyIncomeCentavos int64 `json:"monthly_income_centavos"`
	GrowthCentavos        int64 `json:"growth_centavos"`
	GrowthBps             int   `json:"growth_bps"`
}

type classSliceResponse struct {
	AssetClass    string `json:"asset_class"`
	ValueCentavos int64  `json:"value_centavos"`
	ShareBps      int    `json:"share_bps"`
}

type sectorSliceResponse struct {
	Sector        string `json:"sector"`
	ValueCentavos int64  `json:"value_centavos"`
	ShareBps      int    `json:"share_bps"`
}

// getDashboard returns the authenticated caller's computed dashboard (SPEC-103 FR-1037).
// Identity comes from the context the middleware set, never from request input (BR-1033).
func (h dashboardHandler) getDashboard(w http.ResponseWriter, r *http.Request) {
	userID, ok := callerID(w, r)
	if !ok {
		return
	}
	d, err := h.service.GetDashboard(r.Context(), userID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "get dashboard failed", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, toDashboardResponse(d))
}

func toDashboardResponse(d dashboard.Dashboard) dashboardResponse {
	allocation := make([]classSliceResponse, len(d.Allocation))
	for i, s := range d.Allocation {
		allocation[i] = classSliceResponse{AssetClass: string(s.Class), ValueCentavos: s.ValueCentavos, ShareBps: s.ShareBps}
	}
	sectors := make([]sectorSliceResponse, len(d.FIISectors))
	for i, s := range d.FIISectors {
		sectors[i] = sectorSliceResponse{Sector: string(s.Sector), ValueCentavos: s.ValueCentavos, ShareBps: s.ShareBps}
	}
	stale := d.StaleTickers
	if stale == nil {
		stale = []string{} // emit [] rather than null
	}
	return dashboardResponse{
		Summary: summaryResponse{
			TotalInvestedCentavos: d.Summary.TotalInvestedCentavos,
			CurrentValueCentavos:  d.Summary.CurrentValueCentavos,
			MonthlyIncomeCentavos: d.Summary.MonthlyIncomeCentavos,
			GrowthCentavos:        d.Summary.GrowthCentavos,
			GrowthBps:             d.Summary.GrowthBps,
		},
		Allocation:   allocation,
		FIISectors:   sectors,
		StaleTickers: stale,
	}
}

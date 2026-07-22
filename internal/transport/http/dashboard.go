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
	// FixedIncomeReconciliationDue/NeedsAttention/FIIHoldings/FixedIncomeHoldings are new in
	// SPEC-110 (FR-1105/FR-1109).
	FixedIncomeReconciliationDue []string                          `json:"fixed_income_reconciliation_due"`
	NeedsAttention               bool                              `json:"needs_attention"`
	FIIHoldings                  []fiiHoldingSliceResponse         `json:"fii_holdings"`
	FixedIncomeHoldings          []fixedIncomeHoldingSliceResponse `json:"fixed_income_holdings"`
}

type summaryResponse struct {
	TotalInvestedCentavos int64 `json:"total_invested_centavos"`
	CurrentValueCentavos  int64 `json:"current_value_centavos"` // the full patrimony / net worth
	MonthlyIncomeCentavos int64 `json:"monthly_income_centavos"`
	GrowthCentavos        int64 `json:"growth_centavos"`
	GrowthBps             int   `json:"growth_bps"`
}

// classSliceResponse's InvestedCentavos/GrowthCentavos/GrowthBps are new in SPEC-110 FR-1104 —
// 0 for Stocks/ETFs (always-zero classes in the MVP).
type classSliceResponse struct {
	AssetClass       string `json:"asset_class"`
	ValueCentavos    int64  `json:"value_centavos"`
	ShareBps         int    `json:"share_bps"`
	InvestedCentavos int64  `json:"invested_centavos"`
	GrowthCentavos   int64  `json:"growth_centavos"`
	GrowthBps        int    `json:"growth_bps"`
}

type sectorSliceResponse struct {
	Sector        string `json:"sector"`
	ValueCentavos int64  `json:"value_centavos"`
	ShareBps      int    `json:"share_bps"`
}

// fiiHoldingSliceResponse / fixedIncomeHoldingSliceResponse are the new per-holding breakdown
// (SPEC-110 FR-1109), mirroring sectorSliceResponse's value+share shape at holding granularity.
type fiiHoldingSliceResponse struct {
	Ticker        string `json:"ticker"`
	ValueCentavos int64  `json:"value_centavos"`
	ShareBps      int    `json:"share_bps"`
}

type fixedIncomeHoldingSliceResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	ValueCentavos  int64  `json:"value_centavos"`
	GrowthCentavos int64  `json:"growth_centavos"`
	ShareBps       int    `json:"share_bps"`
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
		allocation[i] = classSliceResponse{
			AssetClass: string(s.Class), ValueCentavos: s.ValueCentavos, ShareBps: s.ShareBps,
			InvestedCentavos: s.InvestedCentavos, GrowthCentavos: s.GrowthCentavos, GrowthBps: s.GrowthBps,
		}
	}
	sectors := make([]sectorSliceResponse, len(d.FIISectors))
	for i, s := range d.FIISectors {
		sectors[i] = sectorSliceResponse{Sector: string(s.Sector), ValueCentavos: s.ValueCentavos, ShareBps: s.ShareBps}
	}
	stale := d.StaleTickers
	if stale == nil {
		stale = []string{} // emit [] rather than null
	}
	dueList := d.FixedIncomeReconciliationDue
	if dueList == nil {
		dueList = []string{}
	}
	fiiHoldings := make([]fiiHoldingSliceResponse, len(d.FIIHoldings))
	for i, s := range d.FIIHoldings {
		fiiHoldings[i] = fiiHoldingSliceResponse{Ticker: s.Ticker.String(), ValueCentavos: s.ValueCentavos, ShareBps: s.ShareBps}
	}
	fiHoldings := make([]fixedIncomeHoldingSliceResponse, len(d.FixedIncomeHoldings))
	for i, s := range d.FixedIncomeHoldings {
		fiHoldings[i] = fixedIncomeHoldingSliceResponse{
			ID: s.ID, Name: s.Name, ValueCentavos: s.ValueCentavos, GrowthCentavos: s.GrowthCentavos, ShareBps: s.ShareBps,
		}
	}
	return dashboardResponse{
		Summary: summaryResponse{
			TotalInvestedCentavos: d.Summary.TotalInvestedCentavos,
			CurrentValueCentavos:  d.Summary.CurrentValueCentavos,
			MonthlyIncomeCentavos: d.Summary.MonthlyIncomeCentavos,
			GrowthCentavos:        d.Summary.GrowthCentavos,
			GrowthBps:             d.Summary.GrowthBps,
		},
		Allocation:                   allocation,
		FIISectors:                   sectors,
		StaleTickers:                 stale,
		FixedIncomeReconciliationDue: dueList,
		NeedsAttention:               d.NeedsAttention,
		FIIHoldings:                  fiiHoldings,
		FixedIncomeHoldings:          fiHoldings,
	}
}

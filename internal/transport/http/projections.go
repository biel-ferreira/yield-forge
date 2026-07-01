package http

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/biel-ferreira/yield-forge/internal/projection"
)

// ProjectionEngine is the slice of the projection service the transport needs (consumer-defined);
// *projection.Service satisfies it (SPEC-107).
type ProjectionEngine interface {
	Project(ctx context.Context, userID string, monthlyContributionCentavos int64, horizonYears int) (projection.Projections, error)
}

type projectionsHandler struct {
	service ProjectionEngine
	logger  *slog.Logger
}

// Query-param bounds (SPEC-107 D7). The contribution defaults to 0, the horizon to 10 years.
const (
	defaultHorizonYears = 10
	minHorizonYears     = 1
	maxHorizonYears     = 40
)

type projectionsResponse struct {
	Income     []incomeScenarioResponse   `json:"income"`
	NetWorth   []netWorthScenarioResponse `json:"net_worth"`
	Disclaimer string                     `json:"disclaimer"`
}

type incomeScenarioResponse struct {
	Scenario        string                `json:"scenario"`
	MonthlyCentavos int64                 `json:"monthly_centavos"`
	AnnualCentavos  int64                 `json:"annual_centavos"`
	Assumptions     incomeAssumptionsResp `json:"assumptions"`
}

type incomeAssumptionsResp struct {
	YieldAdjBps int    `json:"yield_adj_bps"`
	Note        string `json:"note"`
}

type netWorthScenarioResponse struct {
	Scenario    string                  `json:"scenario"`
	Points      []netWorthPointResponse `json:"points"`
	Assumptions netWorthAssumptionsResp `json:"assumptions"`
}

type netWorthPointResponse struct {
	Year          int   `json:"year"`
	ValueCentavos int64 `json:"value_centavos"`
}

type netWorthAssumptionsResp struct {
	YieldAdjBps                 int    `json:"yield_adj_bps"`
	MonthlyContributionCentavos int64  `json:"monthly_contribution_centavos"`
	HorizonYears                int    `json:"horizon_years"`
	Note                        string `json:"note"`
}

// getProjections returns the caller's income + net-worth projections (SPEC-107 FR-1075). Identity
// comes from the context (BR-1074); the only client inputs are the contribution + horizon query
// params (integers, never float; bounded), parsed before any computation.
func (h projectionsHandler) getProjections(w http.ResponseWriter, r *http.Request) {
	userID, ok := callerID(w, r)
	if !ok {
		return
	}
	contribution, horizon, ok := parseProjectionParams(w, r)
	if !ok {
		return
	}
	ps, err := h.service.Project(r.Context(), userID, contribution, horizon)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "projections failed", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, toProjectionsResponse(ps))
}

// parseProjectionParams reads the contribution (≥ 0) + horizon (1–40) from the query, applying
// defaults and rejecting non-integer / out-of-range values with a 400 (writes the error itself).
func parseProjectionParams(w http.ResponseWriter, r *http.Request) (contribution int64, horizon int, ok bool) {
	q := r.URL.Query()

	contribution = 0
	if v := q.Get("monthly_contribution_centavos"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, "monthly_contribution_centavos must be a non-negative integer")
			return 0, 0, false
		}
		contribution = n
	}

	horizon = defaultHorizonYears
	if v := q.Get("horizon_years"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < minHorizonYears || n > maxHorizonYears {
			writeError(w, http.StatusBadRequest, "horizon_years must be an integer between 1 and 40")
			return 0, 0, false
		}
		horizon = n
	}
	return contribution, horizon, true
}

func toProjectionsResponse(ps projection.Projections) projectionsResponse {
	income := make([]incomeScenarioResponse, len(ps.Income))
	for i, s := range ps.Income {
		income[i] = incomeScenarioResponse{
			Scenario: string(s.Scenario), MonthlyCentavos: s.MonthlyCentavos, AnnualCentavos: s.AnnualCentavos,
			Assumptions: incomeAssumptionsResp{YieldAdjBps: s.Assumptions.YieldAdjBps, Note: s.Assumptions.Note},
		}
	}
	netWorth := make([]netWorthScenarioResponse, len(ps.NetWorth))
	for i, s := range ps.NetWorth {
		points := make([]netWorthPointResponse, len(s.Points))
		for j, p := range s.Points {
			points[j] = netWorthPointResponse{Year: p.Year, ValueCentavos: p.ValueCentavos}
		}
		netWorth[i] = netWorthScenarioResponse{
			Scenario: string(s.Scenario), Points: points,
			Assumptions: netWorthAssumptionsResp{
				YieldAdjBps: s.Assumptions.YieldAdjBps, MonthlyContributionCentavos: s.Assumptions.MonthlyContributionCentavos,
				HorizonYears: s.Assumptions.HorizonYears, Note: s.Assumptions.Note,
			},
		}
	}
	return projectionsResponse{Income: income, NetWorth: netWorth, Disclaimer: ps.Disclaimer}
}

package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/biel-ferreira/yield-forge/internal/rebalancing"
)

// RebalancingEngine is the slice of the rebalancing engine the transport needs (consumer-defined);
// *rebalancing.Service satisfies it (SPEC-105).
type RebalancingEngine interface {
	Rebalance(ctx context.Context, userID string, contribution rebalancing.Contribution, opts rebalancing.Options) (rebalancing.Rebalancing, error)
}

type rebalancingHandler struct {
	service RebalancingEngine
	logger  *slog.Logger
}

// rebalancingRequest is the contribution input. Money is an integer field, so a float body is
// rejected by the decoder — the float ban holds at the edge (SPEC-105 BR-1051).
type rebalancingRequest struct {
	ContributionCentavos int64 `json:"contribution_centavos"`
	IncludeAssetShares   bool  `json:"include_asset_shares"`
}

type rebalancingResponse struct {
	Areas      []areaResponse `json:"areas"`
	Disclaimer string         `json:"disclaimer"`
	Available  bool           `json:"available"` // false => the LLM was fully unavailable
}

type areaResponse struct {
	Class                   string              `json:"class"`
	SuggestedShareBps       int                 `json:"suggested_share_bps"`
	SuggestedAmountCentavos int64               `json:"suggested_amount_centavos"`
	Title                   string              `json:"title"`
	Detail                  string              `json:"detail"`
	Explanation             string              `json:"explanation"` // always present — the explainability gate guarantees it
	Candidates              []candidateResponse `json:"candidates"`
}

type candidateResponse struct {
	Ticker               string `json:"ticker"`
	Sector               string `json:"sector"`
	Title                string `json:"title"`
	Detail               string `json:"detail"`
	Explanation          string `json:"explanation"`
	IllustrativeShareBps int    `json:"illustrative_share_bps,omitempty"` // only when include_asset_shares
}

// postRebalancing returns the caller's contribution guidance (SPEC-105 FR-1056). Identity comes
// from the context (BR-1055); the contribution is the only client input, parsed > 0 (BR-1051).
// Empty portfolio and LLM-degraded states are both `200` (FR-1057) — `available` distinguishes them.
func (h rebalancingHandler) postRebalancing(w http.ResponseWriter, r *http.Request) {
	userID, ok := callerID(w, r)
	if !ok {
		return
	}
	var req rebalancingRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	contribution, err := rebalancing.ParseContribution(req.ContributionCentavos)
	if err != nil {
		writeError(w, http.StatusBadRequest, "contribution_centavos must be a positive integer")
		return
	}

	res, err := h.service.Rebalance(r.Context(), userID, contribution, rebalancing.Options{IncludeAssetShares: req.IncludeAssetShares})
	if err != nil {
		h.logger.ErrorContext(r.Context(), "rebalance failed", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, toRebalancingResponse(res))
}

// toRebalancingResponse nests the grounded candidates inside the FII area (D6) — they are all FII,
// so they belong to that area; when FIIs are not a suggested area, the candidates are not shown.
func toRebalancingResponse(res rebalancing.Rebalancing) rebalancingResponse {
	areas := make([]areaResponse, 0, len(res.Areas))
	for _, a := range res.Areas {
		ar := areaResponse{
			Class:                   a.Class,
			SuggestedShareBps:       a.SuggestedShareBps,
			SuggestedAmountCentavos: a.SuggestedAmountCentavos,
			Title:                   a.Title,
			Detail:                  a.Detail,
			Explanation:             a.Explanation,
			Candidates:              []candidateResponse{},
		}
		if a.Class == "fii" {
			for _, c := range res.Candidates {
				ar.Candidates = append(ar.Candidates, candidateResponse{
					Ticker: c.Ticker, Sector: c.Sector, Title: c.Title, Detail: c.Detail,
					Explanation: c.Explanation, IllustrativeShareBps: c.IllustrativeShareBps,
				})
			}
		}
		areas = append(areas, ar)
	}
	return rebalancingResponse{Areas: areas, Disclaimer: res.Disclaimer, Available: res.Available}
}

package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/biel-ferreira/yield-forge/internal/health"
)

// HealthScorer is the slice of the health service the transport needs (consumer-defined);
// *health.Service satisfies it (SPEC-106).
type HealthScorer interface {
	Score(ctx context.Context, userID string) (health.HealthScore, error)
}

type healthHandler struct {
	service HealthScorer
	logger  *slog.Logger
}

type healthResponse struct {
	Score              int                    `json:"score"` // 0–100, computed (never LLM)
	Factors            []healthFactorResponse `json:"factors"`
	Narrative          string                 `json:"narrative"`           // optional AI prose; "" when unavailable
	NarrativeAvailable bool                   `json:"narrative_available"` // false on a full LLM outage
	Disclaimer         string                 `json:"disclaimer"`          // present with the narrative (FR-014)
}

type healthFactorResponse struct {
	Name        string `json:"name"`
	Score       int    `json:"score"`      // 0–100
	WeightBps   int    `json:"weight_bps"` // factor weight, basis points (factors sum to 10000)
	Explanation string `json:"explanation"`
}

// getHealthScore returns the caller's reproducible Portfolio Health Score + the gated narrative
// (SPEC-106 FR-1065). Identity comes from the context (BR-1064), never request input. The score +
// breakdown are always present; the narrative degrades to empty on an LLM outage.
func (h healthHandler) getHealthScore(w http.ResponseWriter, r *http.Request) {
	userID, ok := callerID(w, r)
	if !ok {
		return
	}
	hs, err := h.service.Score(r.Context(), userID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "health score failed", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, toHealthResponse(hs))
}

func toHealthResponse(hs health.HealthScore) healthResponse {
	factors := make([]healthFactorResponse, len(hs.Factors))
	for i, f := range hs.Factors {
		factors[i] = healthFactorResponse{
			Name: string(f.Factor), Score: f.Score, WeightBps: f.WeightBps, Explanation: f.Explanation,
		}
	}
	return healthResponse{
		Score: hs.Score, Factors: factors,
		Narrative: hs.Narrative, NarrativeAvailable: hs.NarrativeAvailable, Disclaimer: hs.Disclaimer,
	}
}

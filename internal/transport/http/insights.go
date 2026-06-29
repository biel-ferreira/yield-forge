package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/biel-ferreira/yield-forge/internal/insight/engine"
)

// InsightsEngine is the slice of the insight engine the transport needs. Consumer-defined here
// so the handler stays testable with a small fake; *engine.Service satisfies it (SPEC-104).
type InsightsEngine interface {
	Insights(ctx context.Context, userID string) (engine.Insights, error)
}

// insightsHandler serves GET /insights.
type insightsHandler struct {
	service InsightsEngine
	logger  *slog.Logger
}

type insightsResponse struct {
	Insights   []insightItemResponse `json:"insights"`
	Disclaimer string                `json:"disclaimer"`
	Available  bool                  `json:"available"` // false => the LLM was fully unavailable
}

type insightItemResponse struct {
	Category    string `json:"category"`
	Title       string `json:"title"`
	Detail      string `json:"detail"`
	Explanation string `json:"explanation"` // always present — the explainability gate guarantees it
}

// getInsights returns the authenticated caller's explainable insights (SPEC-104 FR-1048).
// Identity comes from the context the middleware set, never from request input (BR-1043).
// Empty portfolio and LLM-degraded states are both `200` (FR-1047, D5) — the body's `available`
// flag + empty `insights` distinguish them for the client.
func (h insightsHandler) getInsights(w http.ResponseWriter, r *http.Request) {
	userID, ok := callerID(w, r)
	if !ok {
		return
	}
	res, err := h.service.Insights(r.Context(), userID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "get insights failed", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, toInsightsResponse(res))
}

func toInsightsResponse(res engine.Insights) insightsResponse {
	items := make([]insightItemResponse, len(res.Items))
	for i, in := range res.Items {
		items[i] = insightItemResponse{
			Category: in.Category, Title: in.Title, Detail: in.Detail, Explanation: in.Explanation,
		}
	}
	return insightsResponse{Insights: items, Disclaimer: res.Disclaimer, Available: res.Available}
}

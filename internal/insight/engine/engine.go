package engine

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/trace"

	"github.com/biel-ferreira/yield-forge/internal/insight"
	"github.com/biel-ferreira/yield-forge/internal/platform/observability"
)

// Service is the insight engine (SPEC-104 FR-1042..1047). It builds one deterministic fact set
// and calls the Insighter once per category, aggregating the GATED results. It depends only on
// the Fact Builder and the Insighter port — user-facing AI text is emitted ONLY through the
// Insighter, so explainability (FR-013) and non-advice (FR-014) hold by construction (BR-1042).
type Service struct {
	facts     *FactBuilder
	insighter insight.Insighter
	tracer    trace.Tracer
}

// NewService builds the engine over the Fact Builder and the (gated) Insighter.
func NewService(facts *FactBuilder, insighter insight.Insighter) *Service {
	return &Service{facts: facts, insighter: insighter, tracer: observability.Tracer("insight")}
}

// Insights builds the caller's facts and generates insights across all categories. An empty
// portfolio short-circuits with no LLM call (FR-1047). Each category is independent: a category
// the Insighter cannot serve (degraded or gate-rejected) is skipped, so a partial result is
// returned when only some succeed; Available is false only when every category failed.
func (s *Service) Insights(ctx context.Context, userID string) (Insights, error) {
	// Span over fact-building for latency visibility. It carries NO fact content — no money,
	// figures, profile, or generated text reach telemetry (BR-505/BR-1046).
	factCtx, span := s.tracer.Start(ctx, "insight.facts")
	facts, err := s.facts.BuildFacts(factCtx, userID)
	span.End()
	if err != nil {
		return Insights{}, fmt.Errorf("insights: %w", err)
	}

	// Empty portfolio: nothing to analyse — return a friendly available-but-empty state, no LLM.
	if !hasHoldings(facts) {
		return Insights{Disclaimer: insight.Disclaimer, Available: true}, nil
	}

	var items []insight.Insight
	succeeded := 0
	for _, c := range AllCategories {
		res, err := s.insighter.Generate(ctx, insight.InsightRequest{
			Facts:  facts,
			Task:   c.Task(),
			UserID: userID,
		})
		if err != nil {
			// A cancelled/timed-out request must abort, not silently degrade to "unavailable".
			if ctx.Err() != nil {
				return Insights{}, fmt.Errorf("insights: %w", ctx.Err())
			}
			continue // this category degraded or was gate-rejected — skip it (FR-1047)
		}
		succeeded++
		for _, in := range res.Insights {
			in.Category = string(c) // tag with the engine's category (the Insighter task)
			items = append(items, in)
		}
	}

	return Insights{
		Items:      items,
		Disclaimer: insight.Disclaimer,
		Available:  succeeded > 0, // false only when every category failed (fully unavailable)
	}, nil
}

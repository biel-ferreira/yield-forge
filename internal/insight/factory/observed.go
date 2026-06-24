package factory

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/biel-ferreira/yield-forge/internal/insight"
	"github.com/biel-ferreira/yield-forge/internal/platform/observability"
)

// observed is the outermost decorator: it opens an "insight.generate" span and counts
// generations by outcome (SPEC-005 FR-508), so AI latency and the insight-generation
// success rate (PRD §10) are measurable. It records metadata only — provider, model,
// task, outcome, cost — and NEVER prompt content, facts, or generated text (BR-505).
// cache_hit is set on the same span by the cached decorator.
//
// Token usage is deferred (free providers, cost 0 — §15); the cost attribute is recorded
// as int64 minor units to fix the shape (CLAUDE.md money convention) for a future paid
// provider.
type observed struct {
	next     insight.Insighter
	provider string
	model    string
	tracer   trace.Tracer
	counter  metric.Int64Counter
}

func newObserved(next insight.Insighter, provider, model string) observed {
	counter, _ := observability.Meter("insight").Int64Counter("insight.generations")
	return observed{
		next:     next,
		provider: provider,
		model:    model,
		tracer:   observability.Tracer("insight"),
		counter:  counter,
	}
}

func (o observed) Generate(ctx context.Context, req insight.InsightRequest) (insight.InsightResult, error) {
	ctx, span := o.tracer.Start(ctx, "insight.generate")
	defer span.End()
	span.SetAttributes(
		attribute.String("insight.provider", o.provider),
		attribute.String("insight.model", o.model),
		attribute.String("insight.task", string(req.Task)),
		attribute.Int64("insight.cost_centavos", 0), // free providers; per-model cost is §15
	)

	res, err := o.next.Generate(ctx, req)

	outcome := classifyOutcome(err)
	span.SetAttributes(attribute.String("insight.outcome", outcome))
	if o.counter != nil {
		o.counter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("insight.provider", o.provider),
			attribute.String("insight.outcome", outcome),
		))
	}
	return res, err
}

// classifyOutcome maps an error to a low-cardinality outcome label.
func classifyOutcome(err error) string {
	switch {
	case err == nil:
		return "success"
	case errors.Is(err, insight.ErrInsufficientFacts):
		return "insufficient_facts"
	case errors.Is(err, insight.ErrMissingExplanation):
		return "rejected_no_explanation"
	case errors.Is(err, insight.ErrAdviceDetected):
		return "rejected_advice"
	case errors.Is(err, insight.ErrInsightsUnavailable):
		return "unavailable"
	default:
		return "error"
	}
}

package rebalancing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	noopTrace "go.opentelemetry.io/otel/trace/noop"

	"github.com/biel-ferreira/yield-forge/internal/insight"
)

func spanRecorder(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(noopTrace.NewTracerProvider())
	})
	return exp
}

// TestRebalance_FactsSpanCarriesNoPII is the FR-1058/BR-505 guarantee: the fact-building span
// exists for latency, but carries no content — no contribution amount, figures, or generated text.
func TestRebalance_FactsSpanCarriesNoPII(t *testing.T) {
	exp := spanRecorder(t)
	ins := &scriptedInsighter{results: []insight.InsightResult{areaExplanation("Renda Fixa"), {}}}
	svc := NewService(fakeFactSource{facts: engineFacts()}, fakeUniverse{quotes: universeHGLG()}, ins)

	_, err := svc.Rebalance(context.Background(), "u1", mustContribution(t, 123_456), Options{})
	require.NoError(t, err)

	var found bool
	for _, span := range exp.GetSpans() {
		if span.Name != "rebalancing.facts" {
			continue
		}
		found = true
		require.Empty(t, span.Attributes, "the facts span carries no content — no amount, figures, or text")
	}
	require.True(t, found, "rebalancing.facts span recorded")
}

package health

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	noopTrace "go.opentelemetry.io/otel/trace/noop"

	"github.com/biel-ferreira/yield-forge/internal/profile"
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

// TestScore_ComputeSpanCarriesNoPII is the FR-1067/BR-505 guarantee: the compute span exists for
// latency, but carries no content — no score, holdings, or profile reach telemetry.
func TestScore_ComputeSpanCarriesNoPII(t *testing.T) {
	exp := spanRecorder(t)
	svc := newService(sampleDashboard(), sampleHoldings(),
		fakeProfile{p: profile.Profile{Risk: profile.RiskModerate}}, fakeMacro{val: 1050, found: true})

	_, err := svc.Score(context.Background(), "u1")
	require.NoError(t, err)

	var found bool
	for _, span := range exp.GetSpans() {
		if span.Name != "health.compute" {
			continue
		}
		found = true
		require.Empty(t, span.Attributes, "the compute span carries no content — no score, holdings, or profile")
	}
	require.True(t, found, "health.compute span recorded")
}

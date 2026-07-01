package projection

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	noopTrace "go.opentelemetry.io/otel/trace/noop"

	"github.com/biel-ferreira/yield-forge/internal/dashboard"
	"github.com/biel-ferreira/yield-forge/internal/portfolio"
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

// TestProject_ComputeSpanCarriesNoPII is the FR-1076/BR-505 guarantee: the compute span exists for
// latency, but carries no content — no figures, holdings, or series reach telemetry.
func TestProject_ComputeSpanCarriesNoPII(t *testing.T) {
	exp := spanRecorder(t)
	svc := NewService(
		fakeDashboard{d: dashboard.Dashboard{Summary: dashboard.Summary{CurrentValueCentavos: 100_000, MonthlyIncomeCentavos: 500}}},
		fakeHoldings{h: portfolio.Holdings{}},
	)

	_, err := svc.Project(context.Background(), "u1", 10_000, 5)
	require.NoError(t, err)

	var found bool
	for _, span := range exp.GetSpans() {
		if span.Name != "projection.compute" {
			continue
		}
		found = true
		require.Empty(t, span.Attributes, "the compute span carries no content — no figures, holdings, or series")
	}
	require.True(t, found, "projection.compute span recorded")
}

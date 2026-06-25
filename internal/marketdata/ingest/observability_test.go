package ingest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	noopTrace "go.opentelemetry.io/otel/trace/noop"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
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

func attr(span tracetest.SpanStub, key string) (string, bool) {
	for _, kv := range span.Attributes {
		if string(kv.Key) == key {
			return kv.Value.Emit(), true
		}
	}
	return "", false
}

// TestRunOnce_SpanCarriesMetadataOnly verifies the ingestion span records low-cardinality
// run metadata (outcome + per-kind counts) and nothing sensitive (SPEC-006 FR-609, BR-608).
func TestRunOnce_SpanCarriesMetadataOnly(t *testing.T) {
	exp := spanRecorder(t)
	w := testWorker(t, marketdata.Fake{}, newMemFIIRepo(), newMemMacroRepo(), "HGLG11")

	w.RunOnce(context.Background())

	spans := exp.GetSpans()
	require.NotEmpty(t, spans)

	var span tracetest.SpanStub
	for _, s := range spans {
		if s.Name == "marketdata.ingest" {
			span = s
		}
	}
	require.Equal(t, "marketdata.ingest", span.Name, "the span is named for the operation, not an id")

	outcome, ok := attr(span, "marketdata.outcome")
	require.True(t, ok)
	require.Equal(t, "success", outcome)
	_, ok = attr(span, "marketdata.fii_ok")
	require.True(t, ok, "per-kind counts are recorded")

	// Only the documented low-cardinality keys appear — no provider token / URL / payload.
	allowed := map[string]bool{
		"marketdata.outcome": true, "marketdata.fii_ok": true, "marketdata.fii_failed": true,
		"marketdata.macro_ok": true, "marketdata.macro_failed": true,
	}
	for _, kv := range span.Attributes {
		require.True(t, allowed[string(kv.Key)], "unexpected span attribute %q (possible leak)", kv.Key)
	}
}

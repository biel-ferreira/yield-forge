package factory

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	noopTrace "go.opentelemetry.io/otel/trace/noop"

	"github.com/biel-ferreira/yield-forge/internal/insight"
	"github.com/biel-ferreira/yield-forge/internal/platform/config"
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

// TestObserved_SpanCarriesMetadataNotContent is the BR-505 guarantee: the AI span
// records provider/model/outcome metadata, but NEVER the facts or any generated content.
func TestObserved_SpanCarriesMetadataNotContent(t *testing.T) {
	exp := spanRecorder(t)

	const secret = "SUPER_SECRET_HOLDING_XYZ"
	cfg := config.Config{InsighterProvider: "fake", InsighterCacheSize: 16, InsighterCacheTTL: time.Hour}
	in := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), &fakeClock{t: time.Unix(1_700_000_000, 0).UTC()})

	_, err := in.Generate(context.Background(), insight.InsightRequest{
		Facts:  insight.Facts{"ticker": secret, "amount_centavos": 123456},
		Task:   "overview",
		UserID: "u1",
	})
	require.NoError(t, err)

	spans := exp.GetSpans()
	require.NotEmpty(t, spans)

	for _, s := range spans {
		require.NotContains(t, s.Name, secret)
		for _, kv := range s.Attributes {
			require.NotContains(t, kv.Value.Emit(), secret, "attribute %q leaked facts content", kv.Key)
			require.NotContains(t, kv.Value.Emit(), "123456", "attribute %q leaked a fact value", kv.Key)
		}
	}

	// The expected low-cardinality metadata IS present.
	span := spans[0]
	require.Equal(t, "insight.generate", span.Name)
	provider, ok := attr(span, "insight.provider")
	require.True(t, ok)
	require.Equal(t, "fake", provider)
	outcome, ok := attr(span, "insight.outcome")
	require.True(t, ok)
	require.Equal(t, "success", outcome)
	cacheHit, ok := attr(span, "insight.cache_hit")
	require.True(t, ok)
	require.Equal(t, "false", cacheHit)
}

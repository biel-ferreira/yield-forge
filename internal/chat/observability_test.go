package chat

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	noopTrace "go.opentelemetry.io/otel/trace/noop"
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

// TestSend_TurnSpanCarriesNoContent is the FR-1089/BR-505 guarantee: the turn span exists for
// latency, but carries no content — no message text, facts, or generated reply reach telemetry.
func TestSend_TurnSpanCarriesNoContent(t *testing.T) {
	exp := spanRecorder(t)
	repo := newFakeRepo()
	svc := newEngine(repo, &fakeContribution{}, &fakeProjection{}, &stubInsighter{result: okResult()})

	_, err := svc.Send(context.Background(), "u1", "", "informação sensível da carteira do usuário")
	require.NoError(t, err)

	var found bool
	for _, span := range exp.GetSpans() {
		if span.Name != "chat.turn" {
			continue
		}
		found = true
		require.Empty(t, span.Attributes, "the turn span carries no content — no message text or facts")
	}
	require.True(t, found, "chat.turn span recorded")
}

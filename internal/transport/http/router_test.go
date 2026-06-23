package http

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	noopTrace "go.opentelemetry.io/otel/trace/noop"

	"github.com/biel-ferreira/yield-forge/internal/auth"
)

// spanRecorder installs an in-memory TracerProvider as the global provider for the
// duration of a test (otelhttp reads the global provider), and restores a no-op
// provider afterwards.
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

func spanAttr(span tracetest.SpanStub, key string) (string, bool) {
	for _, kv := range span.Attributes {
		if string(kv.Key) == key {
			return kv.Value.AsString(), true
		}
	}
	return "", false
}

func TestHTTP_ServerSpanIsRouteNamed(t *testing.T) {
	exp := spanRecorder(t)

	user := auth.User{ID: "u1", Email: "me@example.com"}
	router := authRouter(fakeAuth{authUser: user, meUser: user})

	rr := doReq(router, http.MethodGet, "/auth/me", "", &http.Cookie{Name: "yf_session", Value: "tok"})
	require.Equal(t, http.StatusOK, rr.Code)

	spans := exp.GetSpans()
	require.Len(t, spans, 1, "one server span per request")
	require.Equal(t, "GET /auth/me", spans[0].Name, "span named by the matched route, not the raw path")

	route, ok := spanAttr(spans[0], "http.route")
	require.True(t, ok, "http.route attribute present")
	require.Equal(t, "GET /auth/me", route)
}

func TestHTTP_ProbesAreNotTraced(t *testing.T) {
	exp := spanRecorder(t)

	router := authRouter(fakeAuth{authErr: auth.ErrSessionNotFound})
	rr := doReq(router, http.MethodGet, "/healthz", "")
	require.Equal(t, http.StatusOK, rr.Code)

	require.Empty(t, exp.GetSpans(), "liveness/readiness probes are filtered out of traces (low-noise)")
}

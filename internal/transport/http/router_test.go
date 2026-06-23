package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
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

func TestHTTP_ContinuesIncomingTrace(t *testing.T) {
	exp := spanRecorder(t)
	// otelhttp uses the global propagator to read an incoming traceparent.
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator()) })

	router := authRouter(fakeAuth{authErr: auth.ErrSessionNotFound})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/version", nil) // public, traced (not a probe)
	// W3C traceparent: version-traceid(32 hex)-spanid(16 hex)-flags
	req.Header.Set("traceparent", "00-0123456789abcdef0123456789abcdef-0123456789abcdef-01")
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	spans := exp.GetSpans()
	require.Len(t, spans, 1)
	require.Equal(t, "0123456789abcdef0123456789abcdef", spans[0].SpanContext.TraceID().String(),
		"the server span continues the incoming trace")
	require.Equal(t, "0123456789abcdef", spans[0].Parent.SpanID().String(),
		"its parent is the incoming span")
}

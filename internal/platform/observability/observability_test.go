package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/biel-ferreira/yield-forge/internal/platform/config"
)

func TestSetup_DisabledIsNoop(t *testing.T) {
	cfg := config.Config{OTELServiceName: "test", OTELExporterKind: "none"}

	shutdown, err := Setup(context.Background(), cfg, "v-test")
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	require.NoError(t, shutdown(context.Background()), "no-op shutdown returns nil")

	// The propagator is installed even when telemetry is disabled, so trace context
	// still flows through the service if an upstream sends it.
	require.NotNil(t, otel.GetTextMapPropagator())
}

func TestSetup_EnabledConstructsAndShutsDown(t *testing.T) {
	t.Cleanup(func() {
		otel.SetTracerProvider(tracenoop.NewTracerProvider())
		otel.SetMeterProvider(metricnoop.NewMeterProvider())
	})

	cfg := config.Config{OTELServiceName: "test", OTELExporterKind: "stdout", OTELTraceSampleRatio: 1.0}

	shutdown, err := Setup(context.Background(), cfg, "v-test")
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	require.NoError(t, shutdown(context.Background()), "shutdown flushes + closes without error")
}

func TestParseHeaders(t *testing.T) {
	got := parseHeaders("authorization=Bearer xyz , x-api-key=abc")
	require.Equal(t, "Bearer xyz", got["authorization"])
	require.Equal(t, "abc", got["x-api-key"])

	require.Empty(t, parseHeaders(""))
	require.Empty(t, parseHeaders("not-a-pair"))
}

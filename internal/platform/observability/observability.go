// Package observability wires OpenTelemetry into the application as cross-cutting
// infrastructure (SPEC-004): a Tracer/Meter pipeline that is safe to run with no
// backend at all. With the exporter kind "none" (the default when no endpoint is
// set), Setup installs only the propagator and leaves the global no-op providers in
// place, so the app behaves identically and pays ~zero overhead (BR-401).
//
// It is platform infrastructure: feature/domain cores import no OTel types; spans and
// metrics are added at the edges (transport, adapters) or via the injected seam
// (SPEC-004 BR-403).
package observability

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/biel-ferreira/yield-forge/internal/platform/config"
)

// noopShutdown is returned when telemetry is disabled — nothing to flush.
func noopShutdown(context.Context) error { return nil }

// Setup installs OpenTelemetry from cfg and returns a shutdown that flushes and closes
// the providers. The W3C propagator is always set (cheap; lets trace context flow even
// when not exporting). When telemetry is disabled (kind "none"), the global providers
// stay the no-op default and shutdown is a no-op. A construction/config error is
// returned for the caller to fail fast; a *backend* being unreachable is not an error
// here (export is lazy + background, BR-401).
func Setup(ctx context.Context, cfg config.Config, version string) (shutdown func(context.Context) error, err error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if !cfg.TelemetryEnabled() {
		return noopShutdown, nil
	}

	res, err := newResource(cfg, version)
	if err != nil {
		return nil, fmt.Errorf("build telemetry resource: %w", err)
	}

	traceExp, metricExp, err := newExporters(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("build telemetry exporters: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.OTELTraceSampleRatio))),
		sdktrace.WithBatcher(traceExp),
	)
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp)),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)

	return func(ctx context.Context) error {
		// Flush + close both providers; surface a combined error.
		return errors.Join(tracerProvider.Shutdown(ctx), meterProvider.Shutdown(ctx))
	}, nil
}

// newResource describes this service for every exported span/metric. Built schemaless
// (well-known attribute keys) so it merges cleanly with the SDK's default resource and
// stays decoupled from any specific semconv version.
func newResource(cfg config.Config, version string) (*resource.Resource, error) {
	return resource.Merge(
		resource.Default(),
		resource.NewSchemaless(
			attribute.String("service.name", cfg.OTELServiceName),
			attribute.String("service.version", version),
			attribute.String("deployment.environment", cfg.AppEnv),
		),
	)
}

// newExporters builds the trace + metric exporters for the configured kind. OTLP uses
// HTTP (not gRPC) to keep the dependency surface lean (SPEC-004 D2); stdout is for dev.
func newExporters(ctx context.Context, cfg config.Config) (sdktrace.SpanExporter, sdkmetric.Exporter, error) {
	switch cfg.OTELExporterKind {
	case "stdout":
		traceExp, err := stdouttrace.New()
		if err != nil {
			return nil, nil, fmt.Errorf("stdout trace exporter: %w", err)
		}
		metricExp, err := stdoutmetric.New()
		if err != nil {
			return nil, nil, fmt.Errorf("stdout metric exporter: %w", err)
		}
		return traceExp, metricExp, nil

	case "otlp":
		headers := parseHeaders(cfg.OTELExporterHeaders)
		traceExp, err := otlptracehttp.New(ctx,
			otlptracehttp.WithEndpointURL(cfg.OTELExporterEndpoint),
			otlptracehttp.WithHeaders(headers),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("otlp trace exporter: %w", err)
		}
		metricExp, err := otlpmetrichttp.New(ctx,
			otlpmetrichttp.WithEndpointURL(cfg.OTELExporterEndpoint),
			otlpmetrichttp.WithHeaders(headers),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("otlp metric exporter: %w", err)
		}
		return traceExp, metricExp, nil

	default:
		return nil, nil, fmt.Errorf("unknown OTEL exporter kind %q", cfg.OTELExporterKind)
	}
}

// parseHeaders parses "k1=v1,k2=v2" into a header map. Malformed pairs are skipped.
func parseHeaders(s string) map[string]string {
	headers := map[string]string{}
	for _, pair := range strings.Split(s, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		k, v, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}
		headers[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return headers
}

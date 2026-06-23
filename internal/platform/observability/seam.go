package observability

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Tracer returns a named Tracer for feature instrumentation (SPEC-004 FR-407). It is
// the seam later specs use: SPEC-005 records AI spans (prompt/model/latency), SPEC-006
// records ingestion spans. When telemetry is disabled this is a no-op tracer.
//
// Conventions for callers (SPEC-004 BR-402/BR-403):
//   - Span names are low-cardinality (operation, not raw ids/values).
//   - Never put secrets or PII (passwords, tokens, raw emails, SQL args) on spans.
//   - Instrument at the edges (adapters/transport) or via this seam — never inside the
//     pure domain core, which imports no OTel types.
//   - End spans with defer span.End().
func Tracer(name string) trace.Tracer { return otel.Tracer(name) }

// Meter returns a named Meter for feature metrics (SPEC-004 FR-407). When telemetry is
// disabled this is a no-op meter.
//
// Money-valued metrics (e.g. the future AI token cost in SPEC-005) follow the project
// money convention: int64 minor units / basis points, never float (CLAUDE.md).
func Meter(name string) metric.Meter { return otel.Meter(name) }

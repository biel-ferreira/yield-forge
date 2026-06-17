// Package logging constructs the application's structured logger.
//
// It builds a *slog.Logger from configuration (level and format), used via
// dependency injection rather than a global. Tracing and metrics are out of
// scope here — they arrive with OpenTelemetry in SPEC-004.
//
// Implemented in SPEC-001 phase 3.
package logging

package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"

	"github.com/biel-ferreira/yield-forge/internal/platform/config"
)

// New builds the application's structured logger from configuration, writing to
// standard output. The returned logger is meant to be injected (passed to the
// components that need it), not stored as a global.
func New(cfg config.Config) *slog.Logger {
	return NewWith(os.Stdout, cfg.LogLevel, cfg.LogFormat)
}

// NewWith builds a structured logger writing to w with the given level and
// format. It is the test seam behind New: tests can capture output by passing a
// buffer. An unknown level falls back to info; an unknown format falls back to
// JSON.
func NewWith(w io.Writer, level, format string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(level)}

	var handler slog.Handler
	if strings.EqualFold(format, "text") {
		handler = slog.NewTextHandler(w, opts)
	} else {
		handler = slog.NewJSONHandler(w, opts)
	}
	// Wrap so every context-aware log call is correlated with the active trace.
	return slog.New(traceHandler{Handler: handler})
}

// traceHandler is a slog.Handler middleware that enriches each record with the active
// span's trace_id and span_id (SPEC-004 FR-405), correlating logs with traces. It is a
// no-op when the context carries no valid span — so logs without a trace (e.g. at
// startup) are unaffected. Correlation only applies to context-aware calls
// (InfoContext/LogAttrs/...); plain Info/Warn without a context cannot be correlated.
type traceHandler struct {
	slog.Handler
}

func (h traceHandler) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.Handler.Handle(ctx, r)
}

// WithAttrs and WithGroup re-wrap so the trace enrichment survives logger.With(...).
func (h traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return traceHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h traceHandler) WithGroup(name string) slog.Handler {
	return traceHandler{Handler: h.Handler.WithGroup(name)}
}

// parseLevel maps a level name to slog.Level, defaulting to info for unknown
// values. Config already normalizes the level, so this is defense-in-depth.
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

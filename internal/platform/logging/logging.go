package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"

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
	return slog.New(handler)
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

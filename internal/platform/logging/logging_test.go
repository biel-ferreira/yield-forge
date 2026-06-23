package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	"github.com/biel-ferreira/yield-forge/internal/platform/config"
)

func TestTraceCorrelation(t *testing.T) {
	traceID, err := trace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
	require.NoError(t, err)
	spanID, err := trace.SpanIDFromHex("0123456789abcdef")
	require.NoError(t, err)
	sc := trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: spanID})

	t.Run("active span adds trace_id and span_id", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewWith(&buf, "info", "json")
		logger.InfoContext(trace.ContextWithSpanContext(context.Background(), sc), "within a span")

		var rec map[string]any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
		require.Equal(t, "0123456789abcdef0123456789abcdef", rec["trace_id"])
		require.Equal(t, "0123456789abcdef", rec["span_id"])
	})

	t.Run("no span leaves the record untouched", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewWith(&buf, "info", "json")
		logger.InfoContext(context.Background(), "no span here")

		require.NotContains(t, buf.String(), "trace_id")
		require.NotContains(t, buf.String(), "span_id")
	})
}

func TestNewWith_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWith(&buf, "info", "json")
	logger.Info("hello", "key", "value")

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
	}
	if rec["msg"] != "hello" {
		t.Errorf("msg = %v, want hello", rec["msg"])
	}
	if rec["level"] != "INFO" {
		t.Errorf("level = %v, want INFO", rec["level"])
	}
	if rec["key"] != "value" {
		t.Errorf("key = %v, want value", rec["key"])
	}
}

func TestNewWith_TextFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWith(&buf, "info", "text")
	logger.Info("hello")

	out := buf.String()
	if !strings.Contains(out, "level=INFO") || !strings.Contains(out, "msg=hello") {
		t.Errorf("text output missing expected fields: %q", out)
	}
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Errorf("expected text format, got JSON-looking output: %q", out)
	}
}

func TestNewWith_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWith(&buf, "warn", "text")
	logger.Debug("dbg")
	logger.Info("inf")
	logger.Warn("wrn")

	out := buf.String()
	if strings.Contains(out, "dbg") {
		t.Errorf("debug should be filtered at warn level: %q", out)
	}
	if strings.Contains(out, "inf") {
		t.Errorf("info should be filtered at warn level: %q", out)
	}
	if !strings.Contains(out, "wrn") {
		t.Errorf("warn should be logged at warn level: %q", out)
	}
}

func TestNewWith_UnknownLevelDefaultsToInfo(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWith(&buf, "bogus", "text")
	logger.Debug("dbg")
	logger.Info("inf")

	out := buf.String()
	if strings.Contains(out, "dbg") {
		t.Errorf("debug should be filtered at the info default: %q", out)
	}
	if !strings.Contains(out, "inf") {
		t.Errorf("info should be logged at the info default: %q", out)
	}
}

func TestNew_ReturnsLogger(t *testing.T) {
	if New(config.Config{LogLevel: "info", LogFormat: "json"}) == nil {
		t.Error("New returned nil")
	}
}

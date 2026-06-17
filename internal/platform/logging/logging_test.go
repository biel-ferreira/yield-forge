package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/biel-ferreira/yield-forge/internal/platform/config"
)

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

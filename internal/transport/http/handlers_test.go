package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/biel-ferreira/yield-forge/internal/platform/buildinfo"
)

// fakePinger is a test double for the readiness dependency: it returns err (nil
// means "healthy") so readiness can be tested without a real database.
type fakePinger struct{ err error }

func (f fakePinger) PingContext(context.Context) error { return f.err }

func testRouter(ready Pinger) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewRouter(logger, buildinfo.Info{
		Version:   "test",
		Commit:    "abc1234",
		BuildTime: "2026-01-01T00:00:00Z",
	}, ready)
}

// doGet issues a GET with a healthy readiness dependency.
func doGet(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	return doGetWith(t, path, fakePinger{})
}

// doGetWith issues a GET with a specific readiness dependency.
func doGetWith(t *testing.T, path string, ready Pinger) *httptest.ResponseRecorder {
	t.Helper()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	testRouter(ready).ServeHTTP(rr, req)
	return rr
}

func TestHealthz(t *testing.T) {
	rr := doGet(t, "/healthz")

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if rr.Header().Get("X-Request-Id") == "" {
		t.Error("X-Request-Id header not set by middleware")
	}
	var body statusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Status != "ok" {
		t.Errorf("status = %q, want ok", body.Status)
	}
}

func TestReadyz_DBUp(t *testing.T) {
	rr := doGetWith(t, "/readyz", fakePinger{}) // healthy

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var body readinessResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Status != "ready" {
		t.Errorf("status = %q, want ready", body.Status)
	}
	if body.Checks["db"] != "up" {
		t.Errorf("checks.db = %q, want up", body.Checks["db"])
	}
}

func TestReadyz_DBDown(t *testing.T) {
	rr := doGetWith(t, "/readyz", fakePinger{err: errors.New("connection refused")})

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rr.Code)
	}
	var body readinessResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Status != "not_ready" {
		t.Errorf("status = %q, want not_ready", body.Status)
	}
	if body.Checks["db"] != "down" {
		t.Errorf("checks.db = %q, want down", body.Checks["db"])
	}
}

func TestVersion(t *testing.T) {
	rr := doGet(t, "/version")

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var body versionResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Version != "test" || body.Commit != "abc1234" || body.BuiltAt != "2026-01-01T00:00:00Z" {
		t.Errorf("version body = %+v, want the injected build info", body)
	}
}

func TestNotFound(t *testing.T) {
	rr := doGet(t, "/does-not-exist")

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json (not HTML default)", ct)
	}
	var body statusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Status != "not found" {
		t.Errorf("status = %q, want 'not found'", body.Status)
	}
}

func TestRequestID_EchoesIncomingHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("X-Request-Id", "my-id-123")

	testRouter(fakePinger{}).ServeHTTP(rr, req)

	if got := rr.Header().Get("X-Request-Id"); got != "my-id-123" {
		t.Errorf("X-Request-Id = %q, want my-id-123 (incoming header echoed)", got)
	}
}

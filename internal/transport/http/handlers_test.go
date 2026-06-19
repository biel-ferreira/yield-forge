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
	"time"

	"github.com/biel-ferreira/yield-forge/internal/auth"
	"github.com/biel-ferreira/yield-forge/internal/platform/buildinfo"
)

// fakePinger is a test double for the readiness dependency: it returns err (nil
// means "healthy") so readiness can be tested without a real database.
type fakePinger struct{ err error }

func (f fakePinger) PingContext(context.Context) error { return f.err }

// fakeAuth is a configurable test double for the auth use cases: each field sets the
// result of the matching method, so handler and middleware tests can drive specific
// outcomes (e.g. a duplicate-email register, a bad-credentials login, an
// authenticated request) without a real service.
type fakeAuth struct {
	registerUser auth.User
	registerErr  error
	loginUser    auth.User
	loginToken   string
	loginErr     error
	logoutErr    error
	authUser     auth.User
	authErr      error
	meUser       auth.User
	meErr        error
}

func (f fakeAuth) Register(context.Context, string, string) (auth.User, error) {
	return f.registerUser, f.registerErr
}
func (f fakeAuth) Login(context.Context, string, string) (auth.User, string, error) {
	return f.loginUser, f.loginToken, f.loginErr
}
func (f fakeAuth) Logout(context.Context, string) error { return f.logoutErr }
func (f fakeAuth) Authenticate(context.Context, string) (auth.User, error) {
	return f.authUser, f.authErr
}
func (f fakeAuth) GetUserByID(context.Context, string) (auth.User, error) {
	return f.meUser, f.meErr
}

func testRouter(ready Pinger) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewRouter(Deps{
		Logger:     logger,
		Build:      buildinfo.Info{Version: "test", Commit: "abc1234", BuildTime: "2026-01-01T00:00:00Z"},
		Ready:      ready,
		Auth:       fakeAuth{authErr: auth.ErrSessionNotFound}, // unauthenticated by default
		CookieName: "yf_session",
		SessionTTL: time.Hour,
	})
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

// TestUnknownRoute_DeniedWithoutSession verifies deny-by-default (SPEC-003 BR-301):
// an unauthenticated request to a non-public route gets a JSON 401, not HTML.
func TestUnknownRoute_DeniedWithoutSession(t *testing.T) {
	rr := doGet(t, "/does-not-exist")

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (deny-by-default)", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json (not HTML default)", ct)
	}
	var body errorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Error == "" {
		t.Error("expected a non-empty error message")
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

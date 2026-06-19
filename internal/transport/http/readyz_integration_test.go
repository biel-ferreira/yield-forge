package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/biel-ferreira/yield-forge/internal/auth"
	"github.com/biel-ferreira/yield-forge/internal/platform/buildinfo"
	"github.com/biel-ferreira/yield-forge/internal/platform/config"
	"github.com/biel-ferreira/yield-forge/internal/platform/database"
)

// TestReadyz_LiveDatabase_Integration exercises /readyz against a real database:
// 200 while the pool is healthy, 503 once it is closed (the dead-DB path the
// handler relies on). Gated like the database integration tests (skips in -short
// mode and without TEST_DATABASE_URL).
func TestReadyz_LiveDatabase_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping readiness integration test in -short mode")
	}
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL (a disposable Postgres) to run this integration test")
	}

	cfg := config.Config{
		DatabaseURL:       url,
		DBMaxOpenConns:    5,
		DBMaxIdleConns:    2,
		DBConnMaxLifetime: 5 * time.Minute,
		DBConnMaxIdleTime: 5 * time.Minute,
		DBConnectTimeout:  5 * time.Second,
	}
	db, err := database.Connect(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouter(Deps{
		Logger:     logger,
		Build:      buildinfo.Info{},
		Ready:      db,
		Auth:       fakeAuth{authErr: auth.ErrSessionNotFound},
		CookieName: "yf_session",
		SessionTTL: time.Hour,
	})

	get := func() *httptest.ResponseRecorder {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		return rr
	}

	// Healthy pool → 200 ready, db up.
	rr := get()
	if rr.Code != http.StatusOK {
		t.Fatalf("readyz (db up) = %d, want 200", rr.Code)
	}
	var up readinessResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &up); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if up.Status != "ready" || up.Checks["db"] != "up" {
		t.Errorf("body = %+v, want ready/db up", up)
	}

	// Close the pool to simulate a dead database → 503 not_ready, db down.
	if err := db.Close(); err != nil {
		t.Fatalf("close pool: %v", err)
	}
	rr = get()
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz (db down) = %d, want 503", rr.Code)
	}
	var down readinessResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &down); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if down.Status != "not_ready" || down.Checks["db"] != "down" {
		t.Errorf("body = %+v, want not_ready/db down", down)
	}
}

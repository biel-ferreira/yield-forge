package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/biel-ferreira/yield-forge/internal/platform/config"
)

// integrationConfig gates a test on a real database: it skips in -short mode and
// when TEST_DATABASE_URL is unset, otherwise returns a Config pointing at it.
//
// TEST_DATABASE_URL must point at a DISPOSABLE Postgres (e.g. the compose db on
// localhost:5433) — the migration round-trip test applies and rolls back schema
// changes against it.
func integrationConfig(t *testing.T) config.Config {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping database integration test in -short mode")
	}
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL (a disposable Postgres) to run database integration tests")
	}
	return config.Config{
		DatabaseURL:       url,
		DBMaxOpenConns:    5,
		DBMaxIdleConns:    2,
		DBConnMaxLifetime: 5 * time.Minute,
		DBConnMaxIdleTime: 5 * time.Minute,
		DBConnectTimeout:  5 * time.Second,
	}
}

func TestConnect_Integration(t *testing.T) {
	cfg := integrationConfig(t)

	db, err := Connect(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(context.Background()); err != nil {
		t.Errorf("PingContext after Connect: %v", err)
	}
}

func TestConnect_UnreachableFailsFast_Integration(t *testing.T) {
	cfg := integrationConfig(t) // gate only; we override the target below
	cfg.DatabaseURL = "postgres://nobody:nobody@127.0.0.1:1/none?sslmode=disable"
	cfg.DBConnectTimeout = 2 * time.Second

	start := time.Now()
	if _, err := Connect(context.Background(), cfg); err == nil {
		t.Fatal("expected an error connecting to an unreachable database")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("Connect blocked for %v; should fail fast within the connect timeout", elapsed)
	}
}

func TestMigrations_RoundTrip_Integration(t *testing.T) {
	url := integrationConfig(t).DatabaseURL

	assertVersion := func(want uint) {
		t.Helper()
		v, dirty, err := MigrationVersion(url)
		if err != nil {
			t.Fatalf("MigrationVersion: %v", err)
		}
		if dirty {
			t.Fatalf("database is dirty at version %d", v)
		}
		if v != want {
			t.Fatalf("schema version = %d, want %d", v, want)
		}
	}

	// up → down → up: proves every migration's down cleanly reverses it (BR-205).
	if err := MigrateUp(url); err != nil {
		t.Fatalf("MigrateUp: %v", err)
	}
	assertVersion(1)

	if err := MigrateDown(url, 1); err != nil {
		t.Fatalf("MigrateDown: %v", err)
	}
	assertVersion(0)

	if err := MigrateUp(url); err != nil {
		t.Fatalf("MigrateUp (again): %v", err)
	}
	assertVersion(1)

	// Re-running up on an up-to-date database is a no-op, not an error (FR-203).
	if err := MigrateUp(url); err != nil {
		t.Fatalf("MigrateUp should be idempotent: %v", err)
	}
	assertVersion(1)
}

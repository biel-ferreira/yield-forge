// Package database provides the application's PostgreSQL connection pool.
//
// It is cross-cutting infrastructure (platform), not tied to any feature: callers
// receive a *sql.DB and inject it where needed (SPEC-002 BR-201). SQL and driver
// types live here and in feature adapter subpackages only — never in a feature core
// (BR-202). The pool is built once in cmd/api and closed on graceful shutdown.
package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/biel-ferreira/yield-forge/internal/platform/config"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
)

// driverName is the database/sql driver registered by the pgx stdlib package.
const driverName = "pgx"

// Connect opens a pooled connection to PostgreSQL using the pgx stdlib driver,
// applies the pool settings from cfg, and verifies connectivity with a bounded
// PingContext (cfg.DBConnectTimeout) before returning. A failure to open or ping is
// returned as an error so the caller can fail fast; the pool is never returned
// half-open (a failed ping closes it first).
func Connect(ctx context.Context, cfg config.Config) (*sql.DB, error) {
	db, err := sql.Open(driverName, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(cfg.DBConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.DBConnMaxIdleTime)

	pingCtx, cancel := context.WithTimeout(ctx, cfg.DBConnectTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping database within %s: %w", cfg.DBConnectTimeout, err)
	}

	return db, nil
}

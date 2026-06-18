package database

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/biel-ferreira/yield-forge/migrations"
)

// withMigrator opens a DEDICATED database connection for migration work — separate
// from the application's request pool — builds a golang-migrate instance over the
// embedded SQL files, runs fn, then closes everything.
//
// Using a private connection (rather than the shared *sql.DB) matters: golang-migrate
// holds a connection for its advisory lock and only releases it on Close. Closing the
// migrator here also closes this dedicated handle, so no connection leaks and there is
// no contention with the pool the HTTP server serves from.
func withMigrator(databaseURL string, fn func(*migrate.Migrate) error) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("load embedded migrations: %w", err)
	}

	db, err := sql.Open(driverName, databaseURL)
	if err != nil {
		return fmt.Errorf("open database for migrations: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("init migrate postgres driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("init migrator: %w", err)
	}
	// m.Close closes both the source and the dedicated database handle above.
	defer m.Close()

	return fn(m)
}

// MigrateUp applies all pending migrations. Re-running against an up-to-date
// database is a no-op (not an error), so the operation is idempotent (FR-203).
func MigrateUp(databaseURL string) error {
	return withMigrator(databaseURL, func(m *migrate.Migrate) error {
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("apply migrations: %w", err)
		}
		return nil
	})
}

// MigrateDown rolls back the given number of migration steps (steps must be >= 1).
// Reaching the bottom with nothing left to roll back is a no-op.
func MigrateDown(databaseURL string, steps int) error {
	if steps < 1 {
		return fmt.Errorf("down steps must be >= 1, got %d", steps)
	}
	return withMigrator(databaseURL, func(m *migrate.Migrate) error {
		if err := m.Steps(-steps); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("roll back %d migration(s): %w", steps, err)
		}
		return nil
	})
}

// MigrationVersion reports the current schema version and whether the database is in
// a dirty (failed-midway) state. A database with no migrations applied yet returns
// version 0, dirty false.
func MigrationVersion(databaseURL string) (version uint, dirty bool, err error) {
	err = withMigrator(databaseURL, func(m *migrate.Migrate) error {
		v, d, verr := m.Version()
		if errors.Is(verr, migrate.ErrNilVersion) {
			version, dirty = 0, false
			return nil
		}
		if verr != nil {
			return fmt.Errorf("read migration version: %w", verr)
		}
		version, dirty = v, d
		return nil
	})
	return version, dirty, err
}

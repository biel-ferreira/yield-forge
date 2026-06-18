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

// newMigrator builds a golang-migrate instance that reads the embedded SQL
// migrations and applies them over the given *sql.DB. Reusing the existing pgx pool
// means no second database driver is needed.
//
// The returned *migrate.Migrate must NOT be Closed here: its database driver wraps
// the shared pool, and closing it would close that pool (owned by the caller).
func newMigrator(db *sql.DB) (*migrate.Migrate, error) {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("load embedded migrations: %w", err)
	}
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("init migrate postgres driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return nil, fmt.Errorf("init migrator: %w", err)
	}
	return m, nil
}

// MigrateUp applies all pending migrations. Re-running against an up-to-date
// database is a no-op (not an error), so the operation is idempotent (FR-203).
func MigrateUp(db *sql.DB) error {
	m, err := newMigrator(db)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// MigrateDown rolls back the given number of migration steps (steps must be >= 1).
// Reaching the bottom with nothing left to roll back is a no-op.
func MigrateDown(db *sql.DB, steps int) error {
	if steps < 1 {
		return fmt.Errorf("down steps must be >= 1, got %d", steps)
	}
	m, err := newMigrator(db)
	if err != nil {
		return err
	}
	if err := m.Steps(-steps); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("roll back %d migration(s): %w", steps, err)
	}
	return nil
}

// MigrationVersion reports the current schema version and whether the database is in
// a dirty (failed-midway) state. A database with no migrations applied yet returns
// version 0, dirty false.
func MigrationVersion(db *sql.DB) (version uint, dirty bool, err error) {
	m, err := newMigrator(db)
	if err != nil {
		return 0, false, err
	}
	v, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("read migration version: %w", err)
	}
	return v, dirty, nil
}

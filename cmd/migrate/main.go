// Command migrate applies, rolls back, and inspects database schema migrations.
//
// Usage:
//
//	go run ./cmd/migrate up                 apply all pending migrations
//	go run ./cmd/migrate down [n]           roll back n migrations (default 1)
//	go run ./cmd/migrate status             print the current schema version
//	go run ./cmd/migrate create <name>      scaffold a new migration pair on disk
//
// up/down/status connect using DATABASE_URL (from the environment or a local .env);
// create is a filesystem-only operation and needs no database.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/biel-ferreira/yield-forge/internal/platform/config"
	"github.com/biel-ferreira/yield-forge/internal/platform/database"
)

// migrationsDir is the on-disk source directory for migration files (create writes
// here; up/down read the copy embedded into the binary).
const migrationsDir = "migrations"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: migrate <up | down [n] | status | create <name>>")
	}

	// create is filesystem-only — handle it before touching the database.
	if args[0] == "create" {
		if len(args) < 2 {
			return errors.New("usage: migrate create <name>")
		}
		return createMigration(args[1])
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, err := database.Connect(context.Background(), cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	switch args[0] {
	case "up":
		if err := database.MigrateUp(db); err != nil {
			return err
		}
		fmt.Println("migrations applied (database is up to date)")
	case "down":
		steps := 1
		if len(args) >= 2 {
			n, err := strconv.Atoi(args[1])
			if err != nil || n < 1 {
				return fmt.Errorf("down step count must be a positive integer, got %q", args[1])
			}
			steps = n
		}
		if err := database.MigrateDown(db, steps); err != nil {
			return err
		}
		fmt.Printf("rolled back %d migration(s)\n", steps)
	case "status":
		v, dirty, err := database.MigrationVersion(db)
		if err != nil {
			return err
		}
		fmt.Printf("schema version: %d (dirty=%t)\n", v, dirty)
	default:
		return fmt.Errorf("unknown command %q (use up | down | status | create)", args[0])
	}
	return nil
}

var seqPrefix = regexp.MustCompile(`^(\d+)_`)

// createMigration scaffolds the next NNNN_<name>.up.sql / .down.sql pair, choosing
// the sequence number as one past the highest existing migration.
func createMigration(name string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read %s: %w", migrationsDir, err)
	}

	maxSeq := 0
	for _, e := range entries {
		if m := seqPrefix.FindStringSubmatch(e.Name()); m != nil {
			if n, _ := strconv.Atoi(m[1]); n > maxSeq {
				maxSeq = n
			}
		}
	}

	next := maxSeq + 1
	slug := slugify(name)
	if slug == "" {
		return errors.New("migration name must contain at least one letter or digit")
	}

	for _, suffix := range []string{"up", "down"} {
		path := filepath.Join(migrationsDir, fmt.Sprintf("%04d_%s.%s.sql", next, slug, suffix))
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists", path)
		}
		header := fmt.Sprintf("-- %04d_%s (%s migration)\n", next, slug, suffix)
		if err := os.WriteFile(path, []byte(header), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		fmt.Println("created", path)
	}
	return nil
}

// slugify lowercases the name and reduces any run of non-alphanumeric characters to
// a single underscore, trimming leading/trailing underscores.
func slugify(name string) string {
	var b strings.Builder
	prevUnderscore := false
	for _, r := range strings.ToLower(name) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevUnderscore = false
		default:
			if !prevUnderscore {
				b.WriteByte('_')
				prevUnderscore = true
			}
		}
	}
	return strings.Trim(b.String(), "_")
}

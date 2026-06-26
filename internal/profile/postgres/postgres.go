// Package postgres implements the profile repository (profile.ProfileRepository) over
// PostgreSQL.
//
// It is an adapter: it depends on the profile core (port + sentinels) and on database/sql,
// never the reverse — the core imports no SQL (SPEC-101, SPEC-002 BR-202). All SQL is
// parameterized and scoped by user_id (BR-1012); the write is an idempotent upsert that
// preserves created_at (BR-1011). Objectives are stored as a jsonb array of the closed enum.
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/biel-ferreira/yield-forge/internal/profile"
)

// invalidTextRepresentation is the SQLSTATE for a malformed value cast (e.g. a non-UUID
// string cast to uuid) — treated as not-found on reads, never a 500.
const invalidTextRepresentation = "22P02"

// Compile-time check that the adapter satisfies the port.
var _ profile.ProfileRepository = ProfileRepository{}

// ProfileRepository is the Postgres-backed profile.ProfileRepository.
type ProfileRepository struct {
	db *sql.DB
}

// NewProfileRepository returns a ProfileRepository over db.
func NewProfileRepository(db *sql.DB) ProfileRepository { return ProfileRepository{db: db} }

// UpsertProfile inserts or replaces the caller's profile and returns the stored row. The
// single ON CONFLICT … RETURNING statement is atomic and idempotent — created_at is left
// untouched on update (so it reflects first creation) while updated_at advances, and the
// RETURNING gives the authoritative timestamps without a second round-trip or a race
// (SPEC-101 FR-1012, BR-1011). The input value objects are already validated, so they are
// returned as-is with the DB-assigned timestamps.
func (r ProfileRepository) UpsertProfile(ctx context.Context, p profile.Profile) (profile.Profile, error) {
	objectives, err := json.Marshal(p.Objectives)
	if err != nil {
		return profile.Profile{}, fmt.Errorf("upsert profile: marshal objectives: %w", err)
	}

	const stmt = `
		INSERT INTO profiles (user_id, risk_profile, objectives, horizon_years, created_at, updated_at)
		VALUES ($1::uuid, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET
			risk_profile  = EXCLUDED.risk_profile,
			objectives    = EXCLUDED.objectives,
			horizon_years = EXCLUDED.horizon_years,
			updated_at    = EXCLUDED.updated_at
		RETURNING created_at, updated_at`

	if err := r.db.QueryRowContext(ctx, stmt,
		p.UserID, string(p.Risk), objectives, p.Horizon.Years(), p.CreatedAt, p.UpdatedAt).
		Scan(&p.CreatedAt, &p.UpdatedAt); err != nil {
		return profile.Profile{}, fmt.Errorf("upsert profile: %w", err)
	}
	return p, nil
}

// GetProfileByUserID returns the profile for userID, or profile.ErrProfileNotFound. The
// stored values are re-validated through their constructors so a corrupt row surfaces as an
// error rather than an invalid value object.
func (r ProfileRepository) GetProfileByUserID(ctx context.Context, userID string) (profile.Profile, error) {
	const q = `
		SELECT user_id::text, risk_profile, objectives, horizon_years, created_at, updated_at
		FROM profiles WHERE user_id = $1::uuid`

	var (
		id             string
		risk           string
		objectivesJSON []byte
		years          int
		p              profile.Profile
	)
	err := r.db.QueryRowContext(ctx, q, userID).Scan(
		&id, &risk, &objectivesJSON, &years, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return profile.Profile{}, profile.ErrProfileNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentation {
		// A malformed user_id can never match a row — treat as not-found.
		return profile.Profile{}, profile.ErrProfileNotFound
	}
	if err != nil {
		return profile.Profile{}, fmt.Errorf("query profile: %w", err)
	}

	rp, err := profile.ParseRiskProfile(risk)
	if err != nil {
		return profile.Profile{}, fmt.Errorf("query profile: %w", err)
	}
	var rawObjectives []string
	if err := json.Unmarshal(objectivesJSON, &rawObjectives); err != nil {
		return profile.Profile{}, fmt.Errorf("query profile: unmarshal objectives: %w", err)
	}
	objectives, err := profile.ParseObjectives(rawObjectives)
	if err != nil {
		return profile.Profile{}, fmt.Errorf("query profile: %w", err)
	}
	horizon, err := profile.ParseHorizon(years)
	if err != nil {
		return profile.Profile{}, fmt.Errorf("query profile: %w", err)
	}

	p.UserID = id
	p.Risk = rp
	p.Objectives = objectives
	p.Horizon = horizon
	return p, nil
}

package postgres_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/platform/config"
	"github.com/biel-ferreira/yield-forge/internal/platform/database"
	"github.com/biel-ferreira/yield-forge/internal/profile"
	"github.com/biel-ferreira/yield-forge/internal/profile/postgres"
)

// profileDB wires the real ProfileRepository against TEST_DATABASE_URL, applies migrations,
// and truncates users (cascading to profiles). Gated like the other integration tests.
func profileDB(t *testing.T) (postgres.ProfileRepository, *sql.DB) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping profile integration test in -short mode")
	}
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL (a disposable Postgres) to run profile integration tests")
	}
	require.NoError(t, database.MigrateUp(url), "apply migrations")

	db, err := database.Connect(context.Background(), config.Config{
		DatabaseURL:       url,
		DBMaxOpenConns:    5,
		DBMaxIdleConns:    2,
		DBConnMaxLifetime: 5 * time.Minute,
		DBConnMaxIdleTime: 5 * time.Minute,
		DBConnectTimeout:  5 * time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.ExecContext(context.Background(), "TRUNCATE users CASCADE")
	require.NoError(t, err)

	return postgres.NewProfileRepository(db), db
}

func createUser(t *testing.T, db *sql.DB, email string) string {
	t.Helper()
	var id string
	err := db.QueryRowContext(context.Background(),
		`INSERT INTO users (email, password_hash) VALUES ($1, 'x') RETURNING id::text`, email).Scan(&id)
	require.NoError(t, err)
	return id
}

func sampleProfile(t *testing.T, userID string, at time.Time) profile.Profile {
	t.Helper()
	h, err := profile.ParseHorizon(10)
	require.NoError(t, err)
	return profile.Profile{
		UserID:     userID,
		Risk:       profile.RiskModerate,
		Objectives: []profile.Objective{profile.ObjectiveRetirement, profile.ObjectivePassiveIncome},
		Horizon:    h,
		CreatedAt:  at,
		UpdatedAt:  at,
	}
}

func TestProfileRepository_UpsertRoundTripAndIdempotency_Integration(t *testing.T) {
	repo, db := profileDB(t)
	ctx := context.Background()
	uid := createUser(t, db, "a@example.com")

	t1 := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	require.NoError(t, repo.UpsertProfile(ctx, sampleProfile(t, uid, t1)))

	got, err := repo.GetProfileByUserID(ctx, uid)
	require.NoError(t, err)
	require.Equal(t, profile.RiskModerate, got.Risk)
	require.Equal(t, []profile.Objective{profile.ObjectiveRetirement, profile.ObjectivePassiveIncome}, got.Objectives)
	require.Equal(t, 10, got.Horizon.Years())
	require.True(t, t1.Equal(got.CreatedAt))

	// Update: a later upsert changes fields, preserves created_at, advances updated_at.
	t2 := t1.Add(48 * time.Hour)
	updated := sampleProfile(t, uid, t2)
	updated.Risk = profile.RiskAggressive
	require.NoError(t, repo.UpsertProfile(ctx, updated))

	got, err = repo.GetProfileByUserID(ctx, uid)
	require.NoError(t, err)
	require.Equal(t, profile.RiskAggressive, got.Risk)
	require.True(t, t1.Equal(got.CreatedAt), "created_at is preserved across upserts (BR-1011)")
	require.True(t, t2.Equal(got.UpdatedAt), "updated_at advances")
}

func TestProfileRepository_NotFound_Integration(t *testing.T) {
	repo, db := profileDB(t)
	uid := createUser(t, db, "b@example.com") // user exists, but has no profile
	_, err := repo.GetProfileByUserID(context.Background(), uid)
	require.ErrorIs(t, err, profile.ErrProfileNotFound)
}

func TestProfileRepository_PerUserIsolation_Integration(t *testing.T) {
	repo, db := profileDB(t)
	ctx := context.Background()
	a := createUser(t, db, "iso-a@example.com")
	b := createUser(t, db, "iso-b@example.com")

	pa := sampleProfile(t, a, time.Now().UTC())
	pa.Risk = profile.RiskConservative
	pb := sampleProfile(t, b, time.Now().UTC())
	pb.Risk = profile.RiskAggressive
	require.NoError(t, repo.UpsertProfile(ctx, pa))
	require.NoError(t, repo.UpsertProfile(ctx, pb))

	gotA, err := repo.GetProfileByUserID(ctx, a)
	require.NoError(t, err)
	require.Equal(t, profile.RiskConservative, gotA.Risk, "user A sees only their own profile")
	require.Equal(t, a, gotA.UserID)
}

func TestProfileRepository_CascadeOnUserDelete_Integration(t *testing.T) {
	repo, db := profileDB(t)
	ctx := context.Background()
	uid := createUser(t, db, "gone@example.com")
	require.NoError(t, repo.UpsertProfile(ctx, sampleProfile(t, uid, time.Now().UTC())))

	_, err := db.ExecContext(ctx, `DELETE FROM users WHERE id = $1::uuid`, uid)
	require.NoError(t, err)

	_, err = repo.GetProfileByUserID(ctx, uid)
	require.ErrorIs(t, err, profile.ErrProfileNotFound, "ON DELETE CASCADE removed the profile")
}

package postgres_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/chat"
	chatpostgres "github.com/biel-ferreira/yield-forge/internal/chat/postgres"
	"github.com/biel-ferreira/yield-forge/internal/platform/config"
	"github.com/biel-ferreira/yield-forge/internal/platform/database"
)

func chatDB(t *testing.T) (*sql.DB, string, string) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping chat integration test in -short mode")
	}
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL (a disposable Postgres) to run chat integration tests")
	}
	require.NoError(t, database.MigrateUp(url), "apply migrations")
	db, err := database.Connect(context.Background(), config.Config{
		DatabaseURL: url, DBMaxOpenConns: 5, DBMaxIdleConns: 2,
		DBConnMaxLifetime: 5 * time.Minute, DBConnMaxIdleTime: 5 * time.Minute, DBConnectTimeout: 5 * time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	_, err = db.ExecContext(context.Background(), "TRUNCATE users CASCADE")
	require.NoError(t, err)

	var u1, u2 string
	require.NoError(t, db.QueryRowContext(context.Background(),
		`INSERT INTO users (email, password_hash) VALUES ('c1@example.com','x') RETURNING id::text`).Scan(&u1))
	require.NoError(t, db.QueryRowContext(context.Background(),
		`INSERT INTO users (email, password_hash) VALUES ('c2@example.com','x') RETURNING id::text`).Scan(&u2))
	return db, u1, u2
}

func TestRepository_RoundTripAndIsolation_Integration(t *testing.T) {
	db, u1, u2 := chatDB(t)
	repo := chatpostgres.New(db)
	ctx := context.Background()
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)

	thr, err := repo.CreateThread(ctx, chat.Thread{UserID: u1, Title: "concentração", CreatedAt: now, UpdatedAt: now})
	require.NoError(t, err)
	require.NotEmpty(t, thr.ID)

	_, err = repo.AppendMessage(ctx, chat.Message{ThreadID: thr.ID, Role: chat.RoleUser, Content: "estou concentrado?", CreatedAt: now})
	require.NoError(t, err)
	_, err = repo.AppendMessage(ctx, chat.Message{ThreadID: thr.ID, Role: chat.RoleAssistant, Content: "Sim, 60%...", Explanation: "porque...", CreatedAt: now.Add(time.Second)})
	require.NoError(t, err)

	msgs, err := repo.ListMessages(ctx, u1, thr.ID)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	require.Equal(t, chat.RoleUser, msgs[0].Role)
	require.Equal(t, chat.RoleAssistant, msgs[1].Role)
	require.Equal(t, "porque...", msgs[1].Explanation)

	// Double-scoping: u2 cannot read u1's thread or its messages.
	_, err = repo.GetThreadByID(ctx, u2, thr.ID)
	require.ErrorIs(t, err, chat.ErrThreadNotFound)
	other, err := repo.ListMessages(ctx, u2, thr.ID)
	require.NoError(t, err)
	require.Empty(t, other, "no cross-user message leak")

	// A malformed (non-UUID) thread id is not-found, not a 500.
	_, err = repo.GetThreadByID(ctx, u1, "not-a-uuid")
	require.ErrorIs(t, err, chat.ErrThreadNotFound)
}

func TestRepository_EnforceCapEvictsOldest_Integration(t *testing.T) {
	db, u1, _ := chatDB(t)
	repo := chatpostgres.New(db)
	ctx := context.Background()
	base := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)

	// Three threads with increasing updated_at.
	var ids []string
	for i := 0; i < 3; i++ {
		at := base.Add(time.Duration(i) * time.Hour)
		thr, err := repo.CreateThread(ctx, chat.Thread{UserID: u1, Title: "t", CreatedAt: at, UpdatedAt: at})
		require.NoError(t, err)
		ids = append(ids, thr.ID)
	}

	// Keep the 2 most recent → the oldest (ids[0]) is evicted.
	require.NoError(t, repo.EnforceCap(ctx, u1, 2))
	remaining, err := repo.ListThreads(ctx, u1)
	require.NoError(t, err)
	require.Len(t, remaining, 2)
	_, err = repo.GetThreadByID(ctx, u1, ids[0])
	require.ErrorIs(t, err, chat.ErrThreadNotFound, "oldest thread evicted")

	// Clear removes everything for the user.
	require.NoError(t, repo.ClearThreads(ctx, u1))
	all, err := repo.ListThreads(ctx, u1)
	require.NoError(t, err)
	require.Empty(t, all)
}

package postgres_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/platform/config"
	"github.com/biel-ferreira/yield-forge/internal/platform/database"
	"github.com/biel-ferreira/yield-forge/internal/portfolio"
	"github.com/biel-ferreira/yield-forge/internal/portfolio/postgres"
)

func portfolioDB(t *testing.T) (postgres.Repository, *sql.DB) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping portfolio integration test in -short mode")
	}
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL (a disposable Postgres) to run portfolio integration tests")
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
	return postgres.New(db), db
}

func createUser(t *testing.T, db *sql.DB, email string) string {
	t.Helper()
	var id string
	err := db.QueryRowContext(context.Background(),
		`INSERT INTO users (email, password_hash) VALUES ($1, 'x') RETURNING id::text`, email).Scan(&id)
	require.NoError(t, err)
	return id
}

func fiiHolding(t *testing.T, userID, ticker string, qty int, priceCentavos int64, at time.Time) portfolio.FIIHolding {
	t.Helper()
	q, err := portfolio.ParseQuantity(qty)
	require.NoError(t, err)
	return portfolio.FIIHolding{
		UserID: userID, Ticker: marketdata.MustParseTicker(ticker), Quantity: q,
		AveragePriceCentavos: priceCentavos, CreatedAt: at, UpdatedAt: at,
	}
}

func TestRepository_FIICRUDRoundTrip_Integration(t *testing.T) {
	repo, db := portfolioDB(t)
	ctx := context.Background()
	uid := createUser(t, db, "a@example.com")
	t1 := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)

	created, err := repo.CreateFIIHolding(ctx, fiiHolding(t, uid, "HGLG11", 100, 15_750, t1))
	require.NoError(t, err)
	require.NotEmpty(t, created.ID, "DB assigns the id")
	require.Equal(t, int64(15_750), created.AveragePriceCentavos)

	list, err := repo.ListFIIHoldingsByUserID(ctx, uid)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "HGLG11", list[0].Ticker.String())
	require.Equal(t, 100, list[0].Quantity.Value())

	// Update: change quantity + price; created_at preserved, updated_at advances.
	upd := created
	upd.AveragePriceCentavos = 16_000
	q, _ := portfolio.ParseQuantity(150)
	upd.Quantity = q
	upd.UpdatedAt = t1.Add(24 * time.Hour)
	got, err := repo.UpdateFIIHolding(ctx, upd)
	require.NoError(t, err)
	require.Equal(t, 150, got.Quantity.Value())
	require.Equal(t, int64(16_000), got.AveragePriceCentavos)
	require.True(t, t1.Equal(got.CreatedAt), "created_at preserved")

	require.NoError(t, repo.DeleteFIIHolding(ctx, uid, created.ID))
	list, err = repo.ListFIIHoldingsByUserID(ctx, uid)
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestRepository_FixedIncomeRoundTrip_Integration(t *testing.T) {
	repo, db := portfolioDB(t)
	ctx := context.Background()
	uid := createUser(t, db, "fi@example.com")
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	mat := time.Date(2030, 1, 15, 0, 0, 0, 0, time.UTC)

	h := portfolio.FixedIncomeHolding{
		UserID: uid, Name: "CDB Liquidez", Institution: "Banco X",
		InvestedAmountCentavos: 1_000_000, AnnualRateBps: 1_250,
		MaturityDate: &mat, LiquidityType: portfolio.LiquidityAtMaturity, CreatedAt: now, UpdatedAt: now,
	}
	created, err := repo.CreateFixedIncomeHolding(ctx, h)
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	list, err := repo.ListFixedIncomeHoldingsByUserID(ctx, uid)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, int64(1_000_000), list[0].InvestedAmountCentavos)
	require.Equal(t, 1_250, list[0].AnnualRateBps)
	require.Equal(t, portfolio.LiquidityAtMaturity, list[0].LiquidityType)
	require.NotNil(t, list[0].MaturityDate)
	require.True(t, mat.Equal(*list[0].MaturityDate))
}

func TestRepository_IsolationAndOwnership_Integration(t *testing.T) {
	repo, db := portfolioDB(t)
	ctx := context.Background()
	a := createUser(t, db, "owner-a@example.com")
	b := createUser(t, db, "owner-b@example.com")
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)

	aHolding, err := repo.CreateFIIHolding(ctx, fiiHolding(t, a, "KNRI11", 50, 14_820, now))
	require.NoError(t, err)

	// Isolation: B sees none of A's holdings.
	bList, err := repo.ListFIIHoldingsByUserID(ctx, b)
	require.NoError(t, err)
	require.Empty(t, bList)

	// Ownership: B cannot update or delete A's holding (scoped by id + user_id).
	steal := aHolding
	steal.UserID = b
	steal.AveragePriceCentavos = 1
	_, err = repo.UpdateFIIHolding(ctx, steal)
	require.ErrorIs(t, err, portfolio.ErrHoldingNotFound, "B cannot update A's holding")

	err = repo.DeleteFIIHolding(ctx, b, aHolding.ID)
	require.ErrorIs(t, err, portfolio.ErrHoldingNotFound, "B cannot delete A's holding")

	// A's holding is untouched.
	aList, err := repo.ListFIIHoldingsByUserID(ctx, a)
	require.NoError(t, err)
	require.Len(t, aList, 1)
	require.Equal(t, int64(14_820), aList[0].AveragePriceCentavos)
}

func TestRepository_NotFoundAndCascade_Integration(t *testing.T) {
	repo, db := portfolioDB(t)
	ctx := context.Background()
	uid := createUser(t, db, "cascade@example.com")
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)

	// Update/delete a non-existent id → not found.
	missing := fiiHolding(t, uid, "HGLG11", 1, 100, now)
	missing.ID = "00000000-0000-0000-0000-000000000000"
	_, err := repo.UpdateFIIHolding(ctx, missing)
	require.ErrorIs(t, err, portfolio.ErrHoldingNotFound)
	require.ErrorIs(t, repo.DeleteFIIHolding(ctx, uid, missing.ID), portfolio.ErrHoldingNotFound)

	// Cascade: deleting the user removes their holdings.
	_, err = repo.CreateFIIHolding(ctx, fiiHolding(t, uid, "MXRF11", 200, 1_030, now))
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `DELETE FROM users WHERE id = $1::uuid`, uid)
	require.NoError(t, err)
	list, err := repo.ListFIIHoldingsByUserID(ctx, uid)
	require.NoError(t, err)
	require.Empty(t, list, "ON DELETE CASCADE removed the holdings")
}

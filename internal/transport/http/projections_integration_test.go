package http

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/dashboard"
	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	marketdatapostgres "github.com/biel-ferreira/yield-forge/internal/marketdata/postgres"
	"github.com/biel-ferreira/yield-forge/internal/platform/config"
	"github.com/biel-ferreira/yield-forge/internal/platform/database"
	"github.com/biel-ferreira/yield-forge/internal/portfolio"
	portfoliopostgres "github.com/biel-ferreira/yield-forge/internal/portfolio/postgres"
	"github.com/biel-ferreira/yield-forge/internal/projection"
)

// This end-to-end projection test lives in the transport layer (where SQL is allowed) because the
// projection core is layering-locked out of database/sql — it exercises the real projection.Service
// over real repositories + Postgres (SPEC-107 §12).

type projClock struct{ t time.Time }

func (c projClock) Now() time.Time { return c.t }

func projectionsDB(t *testing.T) *sql.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping projection integration test in -short mode")
	}
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL (a disposable Postgres) to run projection integration tests")
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
	_, err = db.ExecContext(context.Background(), "TRUNCATE fii_quotes, macro_indicators")
	require.NoError(t, err)
	return db
}

func baseIncome(ps projection.Projections) projection.ScenarioIncome {
	for _, i := range ps.Income {
		if i.Scenario == projection.ScenarioBase {
			return i
		}
	}
	return projection.ScenarioIncome{}
}

// TestProject_ReproducibleEndToEnd is the spec's key proof (SPEC-107 §12): real Postgres, seeded
// across SPEC-102/006. The base income reconciles with the seeded holdings/market, a positive
// scenario's net-worth series is monotonic, TWO calls return an identical projection
// (reproducibility), and one user's holdings never reach another (per-user scoping).
func TestProject_ReproducibleEndToEnd_Integration(t *testing.T) {
	db := projectionsDB(t)
	ctx := context.Background()
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	clk := projClock{t: now}

	var u1, u2 string
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash) VALUES ('pr1@example.com','x') RETURNING id::text`).Scan(&u1))
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash) VALUES ('pr2@example.com','x') RETURNING id::text`).Scan(&u2))

	pfRepo := portfoliopostgres.New(db)
	quoteRepo := marketdatapostgres.NewFIIQuoteRepository(db)
	macroRepo := marketdatapostgres.NewMacroRepository(db)

	qty, err := portfolio.ParseQuantity(100)
	require.NoError(t, err)
	_, err = pfRepo.CreateFIIHolding(ctx, portfolio.FIIHolding{
		UserID: u1, Ticker: marketdata.MustParseTicker("HGLG11"), Quantity: qty,
		AveragePriceCentavos: 15_750, CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)
	_, err = pfRepo.CreateFixedIncomeHolding(ctx, portfolio.FixedIncomeHolding{
		UserID: u1, Name: "CDB", Institution: "Banco", InvestedAmountCentavos: 1_000_000,
		AnnualRateBps: 1_200, LiquidityType: portfolio.LiquidityDaily, CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)
	require.NoError(t, quoteRepo.UpsertFIIQuote(ctx, marketdata.FIIQuote{
		Ticker: marketdata.MustParseTicker("HGLG11"), PriceCentavos: 16_000, DividendYieldBps: 850,
		Sector: marketdata.SectorLogistics, LastDividendCentavos: 110, Source: "test", ObservedAt: now, FetchedAt: now,
	}))

	dashSvc := dashboard.NewService(portfolio.NewService(pfRepo, clk, macroRepo), quoteRepo, clk)
	svc := projection.NewService(dashSvc, portfolio.NewService(pfRepo, clk, macroRepo))

	got, err := svc.Project(ctx, u1, 50_000, 10)
	require.NoError(t, err)

	// Base income reconciles: FII monthly 110×100 = 11_000 → 132_000/yr; FI 12%×1M = 120_000/yr.
	require.Equal(t, int64(252_000), baseIncome(got).AnnualCentavos)
	require.NotEmpty(t, got.Disclaimer)

	// The optimistic net-worth series is monotonic across year 0..10.
	var opt projection.ScenarioNetWorth
	for _, n := range got.NetWorth {
		if n.Scenario == projection.ScenarioOptimistic {
			opt = n
		}
	}
	require.Len(t, opt.Points, 11)
	for i := 1; i < len(opt.Points); i++ {
		require.Greater(t, opt.Points[i].ValueCentavos, opt.Points[i-1].ValueCentavos)
	}

	// Reproducibility end to end: a second call returns the identical projection.
	again, err := svc.Project(ctx, u1, 50_000, 10)
	require.NoError(t, err)
	require.Equal(t, got, again)

	// Per-user isolation: u2 has no holdings → zero income, no leak of u1's portfolio.
	other, err := svc.Project(ctx, u2, 50_000, 10)
	require.NoError(t, err)
	require.Equal(t, int64(0), baseIncome(other).AnnualCentavos)
}

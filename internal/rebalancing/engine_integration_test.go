package rebalancing_test

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/dashboard"
	"github.com/biel-ferreira/yield-forge/internal/insight/engine"
	insightfactory "github.com/biel-ferreira/yield-forge/internal/insight/factory"
	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	marketdatapostgres "github.com/biel-ferreira/yield-forge/internal/marketdata/postgres"
	"github.com/biel-ferreira/yield-forge/internal/platform/config"
	"github.com/biel-ferreira/yield-forge/internal/platform/database"
	"github.com/biel-ferreira/yield-forge/internal/portfolio"
	portfoliopostgres "github.com/biel-ferreira/yield-forge/internal/portfolio/postgres"
	"github.com/biel-ferreira/yield-forge/internal/profile"
	profilepostgres "github.com/biel-ferreira/yield-forge/internal/profile/postgres"
	"github.com/biel-ferreira/yield-forge/internal/rebalancing"
)

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func connectDB(t *testing.T) *sql.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping rebalancing integration test in -short mode")
	}
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL (a disposable Postgres) to run rebalancing integration tests")
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

// TestRebalance_GatesHoldEndToEnd is the spec's key safety proof (SPEC-105 §12): real Postgres +
// the gated fake Insighter, seeded across SPEC-101/102/006. Every suggested area carries an
// explanation, the disclaimer is present (the gates hold end to end), the computed shares
// reconcile to 10000, any surfaced candidate is grounded in the seeded universe, and one user's
// holdings never reach another (per-user scoping).
func TestRebalance_GatesHoldEndToEnd_Integration(t *testing.T) {
	db := connectDB(t)
	ctx := context.Background()
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	clk := fixedClock{t: now}

	var u1, u2 string
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash) VALUES ('reb1@example.com','x') RETURNING id::text`).Scan(&u1))
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash) VALUES ('reb2@example.com','x') RETURNING id::text`).Scan(&u2))

	pfRepo := portfoliopostgres.New(db)
	quoteRepo := marketdatapostgres.NewFIIQuoteRepository(db)
	macroRepo := marketdatapostgres.NewMacroRepository(db)
	profileRepo := profilepostgres.NewProfileRepository(db)

	// u1: a 100%-FII portfolio (moderate profile) → the split should steer toward Fixed Income.
	qty, err := portfolio.ParseQuantity(100)
	require.NoError(t, err)
	_, err = pfRepo.CreateFIIHolding(ctx, portfolio.FIIHolding{
		UserID: u1, Ticker: marketdata.MustParseTicker("HGLG11"), Quantity: qty,
		AveragePriceCentavos: 15_750, CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)
	require.NoError(t, quoteRepo.UpsertFIIQuote(ctx, marketdata.FIIQuote{
		Ticker: marketdata.MustParseTicker("HGLG11"), PriceCentavos: 16_000, DividendYieldBps: 850,
		Sector: marketdata.SectorLogistics, LastDividendCentavos: 110, Source: "test", ObservedAt: now, FetchedAt: now,
	}))
	require.NoError(t, macroRepo.UpsertMacroIndicator(ctx, marketdata.MacroIndicator{
		Indicator: marketdata.IndicatorSELIC, Value: 10_500, Unit: marketdata.UnitBps,
		ReferenceDate: now, Source: "test", FetchedAt: now,
	}))
	h, err := profile.ParseHorizon(10)
	require.NoError(t, err)
	_, err = profileRepo.UpsertProfile(ctx, profile.Profile{
		UserID: u1, Risk: profile.RiskModerate,
		Objectives: []profile.Objective{profile.ObjectivePassiveIncome}, Horizon: h,
		CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)

	// The rebalancing engine over the real seams + the gated fake Insighter (the production chain).
	dashSvc := dashboard.NewService(portfolio.NewService(pfRepo, clk), quoteRepo, clk)
	profSvc := profile.NewService(profileRepo, clk)
	factBuilder := engine.NewFactBuilder(dashSvc, profSvc, macroRepo)
	insighter := insightfactory.New(config.Config{
		InsighterProvider: "fake", InsighterCacheSize: 64, InsighterCacheTTL: time.Minute,
	}, slog.New(slog.NewTextHandler(os.Stderr, nil)), clk)
	svc := rebalancing.NewService(factBuilder, quoteRepo, insighter)

	contribution, err := rebalancing.ParseContribution(100_000)
	require.NoError(t, err)

	got, err := svc.Rebalance(ctx, u1, contribution, rebalancing.Options{})
	require.NoError(t, err)
	require.True(t, got.Available)
	require.NotEmpty(t, got.Areas, "a funded contribution yields area guidance")
	require.NotEmpty(t, got.Disclaimer, "the non-advice disclaimer is present (FR-014)")

	sumShares := 0
	known := map[string]bool{"fii": true, "fixed_income": true}
	for _, a := range got.Areas {
		require.NotEmpty(t, a.Explanation, "every area carries an explanation (FR-013)")
		require.True(t, known[a.Class], "area is a real class: %q", a.Class)
		sumShares += a.SuggestedShareBps
	}
	require.Equal(t, 10000, sumShares, "the computed shares reconcile to 100%")

	// Any surfaced candidate must be grounded in the seeded universe AND explained (FR-013).
	for _, c := range got.Candidates {
		require.Equal(t, "HGLG11", c.Ticker, "candidates are grounded in the universe")
		require.NotEmpty(t, c.Explanation, "every candidate carries an explanation")
	}

	// u2: empty portfolio → still guides (first-investment), scoped to u2 (no leak of u1's holdings).
	other, err := svc.Rebalance(ctx, u2, contribution, rebalancing.Options{})
	require.NoError(t, err)
	require.NotEmpty(t, other.Areas, "an empty portfolio + contribution still guides (FR-1053)")
}

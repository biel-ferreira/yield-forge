package health_test

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/dashboard"
	"github.com/biel-ferreira/yield-forge/internal/health"
	insightfactory "github.com/biel-ferreira/yield-forge/internal/insight/factory"
	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	marketdatapostgres "github.com/biel-ferreira/yield-forge/internal/marketdata/postgres"
	"github.com/biel-ferreira/yield-forge/internal/platform/config"
	"github.com/biel-ferreira/yield-forge/internal/platform/database"
	"github.com/biel-ferreira/yield-forge/internal/portfolio"
	portfoliopostgres "github.com/biel-ferreira/yield-forge/internal/portfolio/postgres"
	"github.com/biel-ferreira/yield-forge/internal/profile"
	profilepostgres "github.com/biel-ferreira/yield-forge/internal/profile/postgres"
)

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func connectDB(t *testing.T) *sql.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping health-score integration test in -short mode")
	}
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL (a disposable Postgres) to run health-score integration tests")
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

// TestScore_ReproducibleEndToEnd is the spec's key proof (SPEC-106 §12): real Postgres + the gated
// fake Insighter, seeded across SPEC-101/102/006. The score is in range, every factor explained,
// TWO calls return an identical score + breakdown (reproducibility), the narrative carries the
// disclaimer, and one user's holdings never reach another (per-user scoping).
func TestScore_ReproducibleEndToEnd_Integration(t *testing.T) {
	db := connectDB(t)
	ctx := context.Background()
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	clk := fixedClock{t: now}

	var u1, u2 string
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash) VALUES ('hs1@example.com','x') RETURNING id::text`).Scan(&u1))
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash) VALUES ('hs2@example.com','x') RETURNING id::text`).Scan(&u2))

	pfRepo := portfoliopostgres.New(db)
	quoteRepo := marketdatapostgres.NewFIIQuoteRepository(db)
	macroRepo := marketdatapostgres.NewMacroRepository(db)
	profileRepo := profilepostgres.NewProfileRepository(db)

	qty, err := portfolio.ParseQuantity(100)
	require.NoError(t, err)
	_, err = pfRepo.CreateFIIHolding(ctx, portfolio.FIIHolding{
		UserID: u1, Ticker: marketdata.MustParseTicker("HGLG11"), Quantity: qty,
		AveragePriceCentavos: 15_750, CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)
	maturity := now.AddDate(2, 0, 0)
	_, err = pfRepo.CreateFixedIncomeHolding(ctx, portfolio.FixedIncomeHolding{
		UserID: u1, Name: "CDB", Institution: "Banco", InvestedAmountCentavos: 1_000_000,
		AnnualRateBps: 1_200, MaturityDate: &maturity, LiquidityType: portfolio.LiquidityAtMaturity,
		CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)
	require.NoError(t, quoteRepo.UpsertFIIQuote(ctx, marketdata.FIIQuote{
		Ticker: marketdata.MustParseTicker("HGLG11"), PriceCentavos: 16_000, DividendYieldBps: 850,
		Sector: marketdata.SectorLogistics, LastDividendCentavos: 110, Source: "test", ObservedAt: now, FetchedAt: now,
	}))
	require.NoError(t, macroRepo.UpsertMacroIndicator(ctx, marketdata.MacroIndicator{
		Indicator: marketdata.IndicatorSELIC, Value: 1050, Unit: marketdata.UnitBps, // 10.50% in bps
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

	svc := health.NewService(
		dashboard.NewService(portfolio.NewService(pfRepo, clk, macroRepo), quoteRepo, clk),
		profile.NewService(profileRepo, clk),
		portfolio.NewService(pfRepo, clk, macroRepo),
		macroRepo,
		insightfactory.New(config.Config{InsighterProvider: "fake", InsighterCacheSize: 64, InsighterCacheTTL: time.Minute},
			slog.New(slog.NewTextHandler(os.Stderr, nil)), clk),
	)

	got, err := svc.Score(ctx, u1)
	require.NoError(t, err)
	require.GreaterOrEqual(t, got.Score, 0)
	require.LessOrEqual(t, got.Score, 100)
	require.Len(t, got.Factors, 5, "all factors present (profile is set)")
	sumWeights := 0
	for _, f := range got.Factors {
		require.NotEmpty(t, f.Explanation, "every factor explained (FR-1062)")
		sumWeights += f.WeightBps
	}
	require.Equal(t, 10000, sumWeights, "weights reconcile")
	require.True(t, got.NarrativeAvailable)
	require.NotEmpty(t, got.Disclaimer, "the narrative carries the non-advice disclaimer")

	// Reproducibility end to end: a second call returns the identical score + breakdown.
	again, err := svc.Score(ctx, u1)
	require.NoError(t, err)
	require.Equal(t, got.Score, again.Score)
	require.Equal(t, got.Factors, again.Factors)

	// Per-user isolation: u2 has no holdings → empty score, no leak of u1's portfolio.
	other, err := svc.Score(ctx, u2)
	require.NoError(t, err)
	require.Equal(t, 0, other.Score, "another user's empty portfolio scores 0 (per-user scoping)")
}

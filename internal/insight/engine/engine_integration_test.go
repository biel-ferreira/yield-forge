package engine_test

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
)

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func connectDB(t *testing.T) *sql.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping insight engine integration test in -short mode")
	}
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL (a disposable Postgres) to run insight engine integration tests")
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
	return db
}

// TestEngine_Insights_GatesHoldEndToEnd is the spec's key safety proof (SPEC-104 §6): real
// Postgres + the gated fake Insighter, seeded across SPEC-101/102/006. Every returned insight
// carries an explanation and the non-advice disclaimer is present — the gates hold end to end —
// and one user's holdings never reach another (per-user scoping).
func TestEngine_Insights_GatesHoldEndToEnd_Integration(t *testing.T) {
	db := connectDB(t)
	ctx := context.Background()
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	clk := fixedClock{t: now}

	// Two users: u1 has a portfolio, u2 is empty (isolation).
	var u1, u2 string
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash) VALUES ('ins1@example.com','x') RETURNING id::text`).Scan(&u1))
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash) VALUES ('ins2@example.com','x') RETURNING id::text`).Scan(&u2))

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
	require.NoError(t, quoteRepo.UpsertFIIQuote(ctx, marketdata.FIIQuote{
		Ticker: marketdata.MustParseTicker("HGLG11"), PriceCentavos: 16_000,
		Sector: marketdata.SectorLogistics, LastDividendCentavos: 110, Source: "test",
		ObservedAt: now, FetchedAt: now,
	}))
	require.NoError(t, macroRepo.UpsertMacroIndicator(ctx, marketdata.MacroIndicator{
		Indicator: marketdata.IndicatorSELIC, Value: 10_500, Unit: marketdata.UnitBps,
		ReferenceDate: now, Source: "test", FetchedAt: now,
	}))
	_, err = profileRepo.UpsertProfile(ctx, profile.Profile{
		UserID: u1, Risk: profile.RiskModerate,
		Objectives: []profile.Objective{profile.ObjectivePassiveIncome}, Horizon: mustHorizon(t, 10),
		CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)

	// The engine over the real seams + the gated fake Insighter (the production chain).
	dashSvc := dashboard.NewService(portfolio.NewService(pfRepo, clk, macroRepo), quoteRepo, clk)
	profSvc := profile.NewService(profileRepo, clk)
	insighter := insightfactory.New(config.Config{
		InsighterProvider: "fake", InsighterCacheSize: 64, InsighterCacheTTL: time.Minute,
	}, slog.New(slog.NewTextHandler(os.Stderr, nil)), clk)
	svc := engine.NewService(engine.NewFactBuilder(dashSvc, profSvc, macroRepo), insighter)

	// u1: gated insights — every item explained, the disclaimer present.
	got, err := svc.Insights(ctx, u1)
	require.NoError(t, err)
	require.True(t, got.Available)
	require.NotEmpty(t, got.Items, "a funded portfolio yields insights")
	require.NotEmpty(t, got.Disclaimer, "the non-advice disclaimer is present (FR-014)")
	valid := map[string]bool{"portfolio": true, "allocation": true, "market_context": true}
	for _, in := range got.Items {
		require.NotEmpty(t, in.Explanation, "every insight carries an explanation (FR-013)")
		require.True(t, valid[in.Category], "insight tagged with a real category: %q", in.Category)
	}

	// u2: empty portfolio — available, no insights, no leakage of u1's holdings.
	other, err := svc.Insights(ctx, u2)
	require.NoError(t, err)
	require.True(t, other.Available)
	require.Empty(t, other.Items, "another user's empty portfolio yields nothing (per-user scoping)")
}

func mustHorizon(t *testing.T, years int) profile.Horizon {
	t.Helper()
	h, err := profile.ParseHorizon(years)
	require.NoError(t, err)
	return h
}

package chat_test

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/chat"
	chatpostgres "github.com/biel-ferreira/yield-forge/internal/chat/postgres"
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
	"github.com/biel-ferreira/yield-forge/internal/projection"
	"github.com/biel-ferreira/yield-forge/internal/rebalancing"
)

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func connectDB(t *testing.T) *sql.DB {
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
	_, err = db.ExecContext(context.Background(), "TRUNCATE fii_quotes, macro_indicators")
	require.NoError(t, err)
	return db
}

// TestChat_GatesHoldEndToEnd is the capstone's key safety proof (SPEC-108 §12): real Postgres + the
// gated fake Insighter, seeded across SPEC-101/102/006. A general turn and a "tenho R$X" turn both
// yield an explained, disclaimer-carrying reply (the gates hold end to end); the thread persists
// ordered messages; and one user's threads never reach another (per-user scoping).
func TestChat_GatesHoldEndToEnd_Integration(t *testing.T) {
	db := connectDB(t)
	ctx := context.Background()
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	clk := fixedClock{t: now}

	var u1, u2 string
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash) VALUES ('chat1@example.com','x') RETURNING id::text`).Scan(&u1))
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash) VALUES ('chat2@example.com','x') RETURNING id::text`).Scan(&u2))

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
		Ticker: marketdata.MustParseTicker("HGLG11"), PriceCentavos: 16_000, DividendYieldBps: 850,
		Sector: marketdata.SectorLogistics, LastDividendCentavos: 110, Source: "test", ObservedAt: now, FetchedAt: now,
	}))
	require.NoError(t, macroRepo.UpsertMacroIndicator(ctx, marketdata.MacroIndicator{
		Indicator: marketdata.IndicatorSELIC, Value: 1_050, Unit: marketdata.UnitBps, ReferenceDate: now, Source: "test", FetchedAt: now,
	}))
	h, err := profile.ParseHorizon(10)
	require.NoError(t, err)
	_, err = profileRepo.UpsertProfile(ctx, profile.Profile{
		UserID: u1, Risk: profile.RiskModerate, Objectives: []profile.Objective{profile.ObjectivePassiveIncome}, Horizon: h,
		CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)

	// The chat engine over the real seams + the gated fake Insighter (the production chain).
	dashSvc := dashboard.NewService(portfolio.NewService(pfRepo, clk, macroRepo), quoteRepo, clk)
	profSvc := profile.NewService(profileRepo, clk)
	factBuilder := engine.NewFactBuilder(dashSvc, profSvc, macroRepo)
	insighter := insightfactory.New(config.Config{InsighterProvider: "fake", InsighterCacheSize: 64, InsighterCacheTTL: time.Minute},
		slog.New(slog.NewTextHandler(os.Stderr, nil)), clk)
	rebal := rebalancing.NewService(factBuilder, quoteRepo, insighter)
	proj := projection.NewService(dashSvc, portfolio.NewService(pfRepo, clk, macroRepo))
	svc := chat.NewService(chatpostgres.New(db), factBuilder, rebal, proj, insighter, clk)

	// Turn 1 — a general question starts a thread.
	r1, err := svc.Send(ctx, u1, "", "estou concentrado demais em logística?")
	require.NoError(t, err)
	require.True(t, r1.Available)
	require.NotEmpty(t, r1.Message.Explanation, "every assistant reply carries an explanation (FR-013)")
	require.NotEmpty(t, r1.Disclaimer, "the non-advice disclaimer is present (FR-014)")
	threadID := r1.Message.ThreadID
	require.NotEmpty(t, threadID)

	// Turn 2 — a contribution turn continues the same thread (grounds via SPEC-105, still gated).
	r2, err := svc.Send(ctx, u1, threadID, "tenho R$2.000 pra aportar esse mês")
	require.NoError(t, err)
	require.Equal(t, threadID, r2.Message.ThreadID)
	require.NotEmpty(t, r2.Message.Explanation)
	require.NotEmpty(t, r2.Disclaimer)

	// The thread persists all four ordered messages (2 user + 2 assistant), each assistant explained.
	thread, msgs, err := svc.Thread(ctx, u1, threadID)
	require.NoError(t, err)
	require.Equal(t, threadID, thread.ID)
	require.Len(t, msgs, 4)
	require.Equal(t, chat.RoleUser, msgs[0].Role)
	require.Equal(t, chat.RoleAssistant, msgs[1].Role)
	require.NotEmpty(t, msgs[1].Explanation)
	require.NotEmpty(t, msgs[3].Explanation)

	// Per-user isolation: u2 opens their own thread and cannot see u1's.
	_, err = svc.Send(ctx, u2, "", "oi")
	require.NoError(t, err)
	_, _, err = svc.Thread(ctx, u2, threadID)
	require.ErrorIs(t, err, chat.ErrThreadNotFound, "u2 cannot read u1's thread")
	u2Threads, err := svc.ListThreads(ctx, u2)
	require.NoError(t, err)
	require.Len(t, u2Threads, 1, "u2 sees only their own thread")

	// Clear removes u1's history.
	require.NoError(t, svc.ClearThreads(ctx, u1))
	u1Threads, err := svc.ListThreads(ctx, u1)
	require.NoError(t, err)
	require.Empty(t, u1Threads)
}

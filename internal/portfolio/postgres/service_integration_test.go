package postgres_test

// This file — unlike postgres_integration_test.go, which tests the Repository directly —
// exercises portfolio.Service against a real Postgres + a real marketdata.MacroRepository. It
// lives in this adapter subpackage (not internal/portfolio/) because the feature-core layering
// hook (block-layering.ps1) forbids a top-level internal/portfolio/*.go file from importing
// database/sql; a service-level *integration* test genuinely needs both the Service (core) and
// the DB, so the adapter subpackage — already exempt, already the home of postgres_integration_test.go —
// is the correct home for it.

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	marketdatapostgres "github.com/biel-ferreira/yield-forge/internal/marketdata/postgres"
	"github.com/biel-ferreira/yield-forge/internal/platform/config"
	"github.com/biel-ferreira/yield-forge/internal/platform/database"
	"github.com/biel-ferreira/yield-forge/internal/portfolio"
	"github.com/biel-ferreira/yield-forge/internal/portfolio/postgres"
)

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func serviceDB(t *testing.T) *sql.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping portfolio service integration test in -short mode")
	}
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL (a disposable Postgres) to run this integration test")
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

// TestService_FixedIncomeIndexer_EffectiveRate_Integration proves FR-1092's own acceptance
// criteria at the layer the HTTP handler actually calls: portfolio.Service (not just the raw
// repository, TestRepository_FixedIncomeIndexer_Integration in postgres_integration_test.go;
// not just the Dashboard's downstream consumption, tested in the dashboard package). Create/
// List/Update must all expose the resolved EffectiveAnnualRateBps against a real, seeded
// marketdata.MacroRepository.
func TestService_FixedIncomeIndexer_EffectiveRate_Integration(t *testing.T) {
	db := serviceDB(t)
	ctx := context.Background()

	var uid string
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash) VALUES ('cdi-svc@example.com','x') RETURNING id::text`).Scan(&uid))

	now := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	// Seeded indicators use a far-future reference_date so GetLatestMacroIndicator's "newest
	// reference_date wins" always picks THIS test's value — this DB is shared with the live
	// yield-forge-api container, whose own (fake-provider) ingestion runs on real wall-clock
	// dates and would otherwise shadow a same-day seed once the sandbox's "today" catches up.
	refDate := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	macroRepo := marketdatapostgres.NewMacroRepository(db)
	// CDI = 10.50% a.a. -> 120% do CDI resolves to 12.60% a.a. (1_260 bps).
	require.NoError(t, macroRepo.UpsertMacroIndicator(ctx, marketdata.MacroIndicator{
		Indicator: marketdata.IndicatorCDI, Value: 1_050, Unit: marketdata.UnitBps,
		ReferenceDate: refDate, Source: "test", FetchedAt: now,
	}))

	svc := portfolio.NewService(postgres.New(db), fixedClock{t: now}, macroRepo)

	created, err := svc.CreateFixedIncomeHolding(ctx, uid, portfolio.FixedIncomeInput{
		Name: "CDB 120% CDI", Institution: "Banco X", InvestedAmountCentavos: 1_000_000,
		AnnualRateBps: 12_000, IndexerType: "cdi_percentual", LiquidityType: "daily",
	})
	require.NoError(t, err)
	require.Equal(t, 12_000, created.AnnualRateBps, "the raw stored value is unchanged")
	require.Equal(t, 1_260, created.EffectiveAnnualRateBps, "Create resolves & exposes the effective rate")

	list, err := svc.ListFixedIncomeHoldings(ctx, uid)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, 1_260, list[0].EffectiveAnnualRateBps, "List resolves & exposes the effective rate")

	// Update: switch to IPCASpread, seed IPCA = 4.50% a.a. -> +5.80% spread resolves to 10.30% a.a.
	require.NoError(t, macroRepo.UpsertMacroIndicator(ctx, marketdata.MacroIndicator{
		Indicator: marketdata.IndicatorIPCA, Value: 450, Unit: marketdata.UnitBps,
		ReferenceDate: refDate, Source: "test", FetchedAt: now,
	}))
	updated, err := svc.UpdateFixedIncomeHolding(ctx, uid, created.ID, portfolio.FixedIncomeInput{
		Name: "CDB 120% CDI", Institution: "Banco X", InvestedAmountCentavos: 1_000_000,
		AnnualRateBps: 580, IndexerType: "ipca_spread", LiquidityType: "daily",
	})
	require.NoError(t, err)
	require.Equal(t, 1_030, updated.EffectiveAnnualRateBps, "Update resolves & exposes the new effective rate")
}

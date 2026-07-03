package dashboard_test

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
)

// fixedClock is a deterministic Clock for the accrual computation.
type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func connectDB(t *testing.T) *sql.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping dashboard integration test in -short mode")
	}
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL (a disposable Postgres) to run dashboard integration tests")
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

// TestService_GetDashboard_ReconcilesEndToEnd seeds real holdings (SPEC-102) and quotes
// (SPEC-006), then asserts the dashboard service computes figures that reconcile across the
// two features — including a stale FII (no quote) and a year of FI accrual.
func TestService_GetDashboard_ReconcilesEndToEnd_Integration(t *testing.T) {
	db := connectDB(t)
	ctx := context.Background()

	var uid string
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash) VALUES ('dash@example.com','x') RETURNING id::text`).Scan(&uid))

	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	pfRepo := portfoliopostgres.New(db)
	quoteRepo := marketdatapostgres.NewFIIQuoteRepository(db)
	macroRepo := marketdatapostgres.NewMacroRepository(db)

	mustQty := func(n int) portfolio.Quantity {
		q, err := portfolio.ParseQuantity(n)
		require.NoError(t, err)
		return q
	}

	// Two FIIs (one will have a quote, one won't) + a fixed-income created a year ago.
	_, err := pfRepo.CreateFIIHolding(ctx, portfolio.FIIHolding{
		UserID: uid, Ticker: marketdata.MustParseTicker("HGLG11"), Quantity: mustQty(100),
		AveragePriceCentavos: 15_750, CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)
	_, err = pfRepo.CreateFIIHolding(ctx, portfolio.FIIHolding{
		UserID: uid, Ticker: marketdata.MustParseTicker("XPLG11"), Quantity: mustQty(10),
		AveragePriceCentavos: 10_000, CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)
	maturity := now.AddDate(2, 0, 0)
	_, err = pfRepo.CreateFixedIncomeHolding(ctx, portfolio.FixedIncomeHolding{
		UserID: uid, Name: "CDB", Institution: "Banco", InvestedAmountCentavos: 1_000_000,
		AnnualRateBps: 1_200, MaturityDate: &maturity, LiquidityType: portfolio.LiquidityAtMaturity,
		CreatedAt: now.AddDate(-1, 0, 0), UpdatedAt: now.AddDate(-1, 0, 0), // created a year ago → 12% accrued
	})
	require.NoError(t, err)

	// Only HGLG11 has a quote; XPLG11 is deliberately stale.
	require.NoError(t, quoteRepo.UpsertFIIQuote(ctx, marketdata.FIIQuote{
		Ticker: marketdata.MustParseTicker("HGLG11"), PriceCentavos: 16_000,
		Sector: marketdata.SectorLogistics, LastDividendCentavos: 110, Source: "test",
		ObservedAt: now, FetchedAt: now,
	}))

	svc := dashboard.NewService(portfolio.NewService(pfRepo, fixedClock{t: now}, macroRepo), quoteRepo, fixedClock{t: now})
	d, err := svc.GetDashboard(ctx, uid)
	require.NoError(t, err)

	require.Equal(t, int64(2_675_000), d.Summary.TotalInvestedCentavos) // 1.575M + 0.1M + 1M
	require.Equal(t, int64(2_820_000), d.Summary.CurrentValueCentavos)  // 1.6M + 0.1M(stale) + 1.12M(FI+accrual)
	require.Equal(t, int64(11_000), d.Summary.MonthlyIncomeCentavos)    // HGLG11 only
	require.Equal(t, []string{"XPLG11"}, d.StaleTickers)

	// Reconciliation across the seeded SPEC-102 + SPEC-006 data.
	var classSum int64
	for _, s := range d.Allocation {
		classSum += s.ValueCentavos
	}
	require.Equal(t, d.Summary.CurrentValueCentavos, classSum, "allocation reconciles to the total")
}

// TestService_GetDashboard_FixedIncomeIndexer_Integration proves SPEC-109 end-to-end: a
// cdi_percentual holding's Dashboard current value reflects the RESOLVED effective rate (seeded
// CDI × the stored percentage), not the raw stored value — the core claim of PLAN-109 Phase 3.
func TestService_GetDashboard_FixedIncomeIndexer_Integration(t *testing.T) {
	db := connectDB(t)
	ctx := context.Background()

	var uid string
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash) VALUES ('cdi-dash@example.com','x') RETURNING id::text`).Scan(&uid))

	created := time.Date(2025, 7, 2, 0, 0, 0, 0, time.UTC)
	now := created.AddDate(1, 0, 0) // exactly 365 days later, for simple round-number accrual

	pfRepo := portfoliopostgres.New(db)
	quoteRepo := marketdatapostgres.NewFIIQuoteRepository(db)
	macroRepo := marketdatapostgres.NewMacroRepository(db)

	// Seed CDI = 10.50% a.a. — the effective rate for "120% do CDI" resolves to 12.60% a.a.
	// A far-future reference_date ensures this test's seed always wins "newest reference_date"
	// against the live yield-forge-api container's own ingestion into this shared dev DB, which
	// runs on real wall-clock dates and would otherwise shadow a same-day seed over time.
	require.NoError(t, macroRepo.UpsertMacroIndicator(ctx, marketdata.MacroIndicator{
		Indicator: marketdata.IndicatorCDI, Value: 1_050, Unit: marketdata.UnitBps,
		ReferenceDate: time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC), Source: "test", FetchedAt: now,
	}))

	_, err := pfRepo.CreateFixedIncomeHolding(ctx, portfolio.FixedIncomeHolding{
		UserID: uid, Name: "CDB 120% CDI", Institution: "Banco X",
		InvestedAmountCentavos: 1_000_000, AnnualRateBps: 12_000, // the RAW stored value: 120.00%
		IndexerType: portfolio.IndexerCDIPercentual, LiquidityType: portfolio.LiquidityDaily,
		CreatedAt: created, UpdatedAt: created,
	})
	require.NoError(t, err)

	svc := dashboard.NewService(portfolio.NewService(pfRepo, fixedClock{t: now}, macroRepo), quoteRepo, fixedClock{t: now})
	d, err := svc.GetDashboard(ctx, uid)
	require.NoError(t, err)

	// Effective rate = 1_050 * 12_000 / 10_000 = 1_260 bps (12.60% a.a.).
	// Accrued over 365 days = 1_000_000 * 1_260 * 365 / (10_000 * 365) = 126_000.
	// If the Dashboard had (incorrectly) used the raw 12_000 bps, current value would be 2_200_000.
	require.Equal(t, int64(1_000_000), d.Summary.TotalInvestedCentavos)
	require.Equal(t, int64(1_126_000), d.Summary.CurrentValueCentavos, "accrual uses the RESOLVED effective rate, not the raw stored value")
}

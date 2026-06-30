package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/marketdata/postgres"
	"github.com/biel-ferreira/yield-forge/internal/platform/config"
	"github.com/biel-ferreira/yield-forge/internal/platform/database"
)

// marketDataDB wires the real Postgres repositories against TEST_DATABASE_URL, applies
// migrations, and truncates the market-data tables for a clean slate. Gated like the other
// integration tests (skips in -short mode and without TEST_DATABASE_URL).
func marketDataDB(t *testing.T) (postgres.FIIQuoteRepository, postgres.MacroRepository) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping market-data integration test in -short mode")
	}
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL (a disposable Postgres) to run market-data integration tests")
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

	_, err = db.ExecContext(context.Background(), "TRUNCATE fii_quotes, macro_indicators")
	require.NoError(t, err)

	return postgres.NewFIIQuoteRepository(db), postgres.NewMacroRepository(db)
}

func TestFIIQuoteRepository_UpsertRoundTripAndIdempotency_Integration(t *testing.T) {
	fii, _ := marketDataDB(t)
	ctx := context.Background()

	lastDiv := time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC)
	q := marketdata.FIIQuote{
		Ticker:               marketdata.MustParseTicker("HGLG11"),
		PriceCentavos:        15_750,
		DividendYieldBps:     850,
		PVPBps:               9_500,
		Sector:               marketdata.SectorLogistics,
		LastDividendCentavos: 110,
		LastDividendDate:     &lastDiv,
		Source:               "fundamentus+yahoo",
		ObservedAt:           time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC),
		FetchedAt:            time.Date(2026, 6, 20, 12, 5, 0, 0, time.UTC),
	}
	require.NoError(t, fii.UpsertFIIQuote(ctx, q))

	got, err := fii.GetFIIQuoteByTicker(ctx, q.Ticker)
	require.NoError(t, err)
	require.Equal(t, q.Ticker, got.Ticker)
	require.Equal(t, q.PriceCentavos, got.PriceCentavos)
	require.Equal(t, q.PVPBps, got.PVPBps)
	require.Equal(t, marketdata.SectorLogistics, got.Sector)
	require.NotNil(t, got.LastDividendDate)
	require.True(t, lastDiv.Equal(*got.LastDividendDate))
	require.True(t, q.FetchedAt.Equal(got.FetchedAt))

	// Idempotent upsert with new values: same row, updated fields (no duplicate).
	q.PriceCentavos = 16_000
	q.DividendYieldBps = 870
	require.NoError(t, fii.UpsertFIIQuote(ctx, q))
	got, err = fii.GetFIIQuoteByTicker(ctx, q.Ticker)
	require.NoError(t, err)
	require.Equal(t, int64(16_000), got.PriceCentavos)
	require.Equal(t, 870, got.DividendYieldBps)
}

func TestFIIQuoteRepository_NotFound_Integration(t *testing.T) {
	fii, _ := marketDataDB(t)
	_, err := fii.GetFIIQuoteByTicker(context.Background(), marketdata.MustParseTicker("XPLG11"))
	require.ErrorIs(t, err, marketdata.ErrFIIQuoteNotFound)
}

func TestFIIQuoteRepository_ListFIIUniverse_Integration(t *testing.T) {
	fii, _ := marketDataDB(t)
	ctx := context.Background()

	// Empty universe before any ingestion → empty slice, not an error (SPEC-105 FR-1054).
	empty, err := fii.ListFIIUniverse(ctx)
	require.NoError(t, err)
	require.Empty(t, empty)

	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	for _, q := range []marketdata.FIIQuote{
		{Ticker: marketdata.MustParseTicker("XPLG11"), PriceCentavos: 10_000, Sector: marketdata.SectorLogistics, Source: "test", ObservedAt: now, FetchedAt: now},
		{Ticker: marketdata.MustParseTicker("HGLG11"), PriceCentavos: 16_000, Sector: marketdata.SectorLogistics, Source: "test", ObservedAt: now, FetchedAt: now},
	} {
		require.NoError(t, fii.UpsertFIIQuote(ctx, q))
	}

	universe, err := fii.ListFIIUniverse(ctx)
	require.NoError(t, err)
	require.Len(t, universe, 2)
	// Ordered by ticker (deterministic).
	require.Equal(t, "HGLG11", universe[0].Ticker.String())
	require.Equal(t, "XPLG11", universe[1].Ticker.String())
	require.Equal(t, marketdata.SectorLogistics, universe[0].Sector)
}

func TestMacroRepository_SeriesLatestAndIdempotency_Integration(t *testing.T) {
	_, macro := marketDataDB(t)
	ctx := context.Background()

	older := marketdata.MacroIndicator{
		Indicator: marketdata.IndicatorSELIC, Value: 10_500, Unit: marketdata.UnitBps,
		ReferenceDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		Source:        "bcb-sgs", FetchedAt: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
	}
	newer := marketdata.MacroIndicator{
		Indicator: marketdata.IndicatorSELIC, Value: 10_250, Unit: marketdata.UnitBps,
		ReferenceDate: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		Source:        "bcb-sgs", FetchedAt: time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC),
	}
	require.NoError(t, macro.UpsertMacroIndicator(ctx, newer))
	require.NoError(t, macro.UpsertMacroIndicator(ctx, older))
	require.NoError(t, macro.UpsertMacroIndicator(ctx, newer)) // re-upsert: idempotent

	got, err := macro.GetLatestMacroIndicator(ctx, marketdata.IndicatorSELIC)
	require.NoError(t, err)
	require.Equal(t, int64(10_250), got.Value, "GetLatest returns the newest reference_date")
	require.True(t, newer.ReferenceDate.Equal(got.ReferenceDate))

	_, err = macro.GetLatestMacroIndicator(ctx, marketdata.IndicatorIFIX)
	require.ErrorIs(t, err, marketdata.ErrMacroNotFound)
}

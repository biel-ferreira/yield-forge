// Package postgres implements the market-data repositories
// (marketdata.FIIQuoteRepository, marketdata.MacroRepository) over PostgreSQL.
//
// It is an adapter: it depends on the marketdata core (ports + sentinels) and on
// database/sql, never the reverse — the core imports no SQL (SPEC-006 BR-601). All SQL is
// parameterized. Writes are idempotent upserts so a re-run, or an overlapping schedule,
// never duplicates or corrupts a row (BR-602). There is no user scoping (BR-603).
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
)

// Compile-time checks that the adapters satisfy the marketdata ports.
var (
	_ marketdata.FIIQuoteRepository = FIIQuoteRepository{}
	_ marketdata.MacroRepository    = MacroRepository{}
)

// FIIQuoteRepository is the Postgres-backed marketdata.FIIQuoteRepository.
type FIIQuoteRepository struct {
	db *sql.DB
}

// NewFIIQuoteRepository returns a FIIQuoteRepository over db.
func NewFIIQuoteRepository(db *sql.DB) FIIQuoteRepository { return FIIQuoteRepository{db: db} }

// UpsertFIIQuote inserts or updates the snapshot for q.Ticker. The single ON CONFLICT
// statement is atomic and idempotent — re-running with the same data yields the same row
// (SPEC-006 FR-605).
func (r FIIQuoteRepository) UpsertFIIQuote(ctx context.Context, q marketdata.FIIQuote) error {
	const stmt = `
		INSERT INTO fii_quotes (
			ticker, price_centavos, dividend_yield_bps, p_vp_bps, sector,
			last_dividend_centavos, last_dividend_date, source, observed_at, fetched_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (ticker) DO UPDATE SET
			price_centavos         = EXCLUDED.price_centavos,
			dividend_yield_bps     = EXCLUDED.dividend_yield_bps,
			p_vp_bps               = EXCLUDED.p_vp_bps,
			sector                 = EXCLUDED.sector,
			last_dividend_centavos = EXCLUDED.last_dividend_centavos,
			last_dividend_date     = EXCLUDED.last_dividend_date,
			source                 = EXCLUDED.source,
			observed_at            = EXCLUDED.observed_at,
			fetched_at             = EXCLUDED.fetched_at`

	var lastDiv sql.NullTime
	if q.LastDividendDate != nil {
		lastDiv = sql.NullTime{Time: *q.LastDividendDate, Valid: true}
	}
	_, err := r.db.ExecContext(ctx, stmt,
		q.Ticker.String(), q.PriceCentavos, q.DividendYieldBps, q.PVPBps, string(q.Sector),
		q.LastDividendCentavos, lastDiv, q.Source, q.ObservedAt, q.FetchedAt)
	if err != nil {
		return fmt.Errorf("upsert fii quote: %w", err)
	}
	return nil
}

// GetFIIQuoteByTicker returns the snapshot for t, or marketdata.ErrFIIQuoteNotFound.
func (r FIIQuoteRepository) GetFIIQuoteByTicker(ctx context.Context, t marketdata.Ticker) (marketdata.FIIQuote, error) {
	const q = `
		SELECT ticker, price_centavos, dividend_yield_bps, p_vp_bps, sector,
			last_dividend_centavos, last_dividend_date, source, observed_at, fetched_at
		FROM fii_quotes WHERE ticker = $1`

	var (
		ticker  string
		sector  string
		lastDiv sql.NullTime
		out     marketdata.FIIQuote
	)
	err := r.db.QueryRowContext(ctx, q, t.String()).Scan(
		&ticker, &out.PriceCentavos, &out.DividendYieldBps, &out.PVPBps, &sector,
		&out.LastDividendCentavos, &lastDiv, &out.Source, &out.ObservedAt, &out.FetchedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return marketdata.FIIQuote{}, marketdata.ErrFIIQuoteNotFound
	}
	if err != nil {
		return marketdata.FIIQuote{}, fmt.Errorf("query fii quote: %w", err)
	}

	parsed, err := marketdata.ParseTicker(ticker)
	if err != nil {
		return marketdata.FIIQuote{}, fmt.Errorf("query fii quote: %w", err)
	}
	out.Ticker = parsed
	out.Sector = marketdata.Sector(sector)
	if lastDiv.Valid {
		d := lastDiv.Time
		out.LastDividendDate = &d
	}
	return out, nil
}

// MacroRepository is the Postgres-backed marketdata.MacroRepository.
type MacroRepository struct {
	db *sql.DB
}

// NewMacroRepository returns a MacroRepository over db.
func NewMacroRepository(db *sql.DB) MacroRepository { return MacroRepository{db: db} }

// UpsertMacroIndicator inserts or updates one observation, idempotent on
// (indicator, reference_date) so re-fetching the same date is a no-op (SPEC-006 FR-605).
func (r MacroRepository) UpsertMacroIndicator(ctx context.Context, m marketdata.MacroIndicator) error {
	const stmt = `
		INSERT INTO macro_indicators (indicator, value, unit, reference_date, source, fetched_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (indicator, reference_date) DO UPDATE SET
			value      = EXCLUDED.value,
			unit       = EXCLUDED.unit,
			source     = EXCLUDED.source,
			fetched_at = EXCLUDED.fetched_at`

	_, err := r.db.ExecContext(ctx, stmt,
		string(m.Indicator), m.Value, string(m.Unit), m.ReferenceDate, m.Source, m.FetchedAt)
	if err != nil {
		return fmt.Errorf("upsert macro indicator: %w", err)
	}
	return nil
}

// GetLatestMacroIndicator returns the most recent observation for ind (newest
// reference_date), or marketdata.ErrMacroNotFound.
func (r MacroRepository) GetLatestMacroIndicator(ctx context.Context, ind marketdata.Indicator) (marketdata.MacroIndicator, error) {
	const q = `
		SELECT indicator, value, unit, reference_date, source, fetched_at
		FROM macro_indicators WHERE indicator = $1
		ORDER BY reference_date DESC LIMIT 1`

	var (
		indicator string
		unit      string
		out       marketdata.MacroIndicator
	)
	err := r.db.QueryRowContext(ctx, q, string(ind)).Scan(
		&indicator, &out.Value, &unit, &out.ReferenceDate, &out.Source, &out.FetchedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return marketdata.MacroIndicator{}, marketdata.ErrMacroNotFound
	}
	if err != nil {
		return marketdata.MacroIndicator{}, fmt.Errorf("query macro indicator: %w", err)
	}
	out.Indicator = marketdata.Indicator(indicator)
	out.Unit = marketdata.Unit(unit)
	return out, nil
}

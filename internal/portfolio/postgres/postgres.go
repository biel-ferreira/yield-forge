// Package postgres implements the portfolio repository (portfolio.Repository) over
// PostgreSQL.
//
// It is an adapter: it depends on the portfolio core (port + sentinels + value objects) and
// on database/sql, never the reverse — the core imports no SQL (SPEC-102, SPEC-002 BR-202).
// All SQL is parameterized. Reads are scoped WHERE user_id = $1; updates and deletes are
// double-scoped WHERE id = $1 AND user_id = $2, so a holding owned by another user is
// indistinguishable from one that does not exist (ErrHoldingNotFound, BR-1021). Money is
// stored as bigint centavos and rates as integer basis points (BR-1022).
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/portfolio"
)

// invalidTextRepresentation is the SQLSTATE for a malformed value cast (e.g. a non-UUID id
// cast to uuid) — treated as not-found on scoped mutations, never a 500.
const invalidTextRepresentation = "22P02"

// Compile-time check that the adapter satisfies the port.
var _ portfolio.Repository = Repository{}

// Repository is the Postgres-backed portfolio.Repository.
type Repository struct {
	db *sql.DB
}

// New returns a Repository over db.
func New(db *sql.DB) Repository { return Repository{db: db} }

// --- FII holdings ---

// CreateFIIHolding inserts a new FII holding and returns it with the DB-assigned id and
// timestamps (SPEC-102 FR-1022).
func (r Repository) CreateFIIHolding(ctx context.Context, h portfolio.FIIHolding) (portfolio.FIIHolding, error) {
	const q = `
		INSERT INTO fii_holdings (user_id, ticker, quantity, average_price_centavos, created_at, updated_at)
		VALUES ($1::uuid, $2, $3, $4, $5, $6)
		RETURNING id::text, created_at, updated_at`
	if err := r.db.QueryRowContext(ctx, q,
		h.UserID, h.Ticker.String(), h.Quantity.Value(), h.AveragePriceCentavos, h.CreatedAt, h.UpdatedAt).
		Scan(&h.ID, &h.CreatedAt, &h.UpdatedAt); err != nil {
		return portfolio.FIIHolding{}, fmt.Errorf("create fii holding: %w", err)
	}
	return h, nil
}

// ListFIIHoldingsByUserID returns the caller's FII holdings (oldest first).
func (r Repository) ListFIIHoldingsByUserID(ctx context.Context, userID string) ([]portfolio.FIIHolding, error) {
	const q = `
		SELECT id::text, user_id::text, ticker, quantity, average_price_centavos, created_at, updated_at
		FROM fii_holdings WHERE user_id = $1::uuid ORDER BY created_at`
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list fii holdings: %w", err)
	}
	defer rows.Close()

	var out []portfolio.FIIHolding
	for rows.Next() {
		var (
			id, uid, ticker  string
			quantity         int
			avgPrice         int64
			created, updated time.Time
		)
		if err := rows.Scan(&id, &uid, &ticker, &quantity, &avgPrice, &created, &updated); err != nil {
			return nil, fmt.Errorf("list fii holdings: %w", err)
		}
		h, err := rebuildFII(id, uid, ticker, quantity, avgPrice, created, updated)
		if err != nil {
			return nil, fmt.Errorf("list fii holdings: %w", err)
		}
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list fii holdings: %w", err)
	}
	return out, nil
}

// UpdateFIIHolding replaces the mutable fields of a holding the caller owns, preserving
// created_at and advancing updated_at. A missing or unowned row → ErrHoldingNotFound.
func (r Repository) UpdateFIIHolding(ctx context.Context, h portfolio.FIIHolding) (portfolio.FIIHolding, error) {
	const q = `
		UPDATE fii_holdings
		SET ticker = $3, quantity = $4, average_price_centavos = $5, updated_at = $6
		WHERE id = $1::uuid AND user_id = $2::uuid
		RETURNING created_at, updated_at`
	err := r.db.QueryRowContext(ctx, q,
		h.ID, h.UserID, h.Ticker.String(), h.Quantity.Value(), h.AveragePriceCentavos, h.UpdatedAt).
		Scan(&h.CreatedAt, &h.UpdatedAt)
	if notFound(err) {
		return portfolio.FIIHolding{}, portfolio.ErrHoldingNotFound
	}
	if err != nil {
		return portfolio.FIIHolding{}, fmt.Errorf("update fii holding: %w", err)
	}
	return h, nil
}

// DeleteFIIHolding removes a holding the caller owns. A missing or unowned row →
// ErrHoldingNotFound.
func (r Repository) DeleteFIIHolding(ctx context.Context, userID, id string) error {
	return r.deleteHolding(ctx, "fii_holdings", userID, id)
}

// --- Fixed income holdings ---

// CreateFixedIncomeHolding inserts a new fixed-income holding and returns it with the
// DB-assigned id and timestamps.
func (r Repository) CreateFixedIncomeHolding(ctx context.Context, h portfolio.FixedIncomeHolding) (portfolio.FixedIncomeHolding, error) {
	const q = `
		INSERT INTO fixed_income_holdings
			(user_id, name, institution, invested_amount_centavos, annual_rate_bps, maturity_date, liquidity_type, created_at, updated_at)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id::text, created_at, updated_at`
	if err := r.db.QueryRowContext(ctx, q,
		h.UserID, h.Name, h.Institution, h.InvestedAmountCentavos, h.AnnualRateBps,
		nullableDate(h.MaturityDate), string(h.LiquidityType), h.CreatedAt, h.UpdatedAt).
		Scan(&h.ID, &h.CreatedAt, &h.UpdatedAt); err != nil {
		return portfolio.FixedIncomeHolding{}, fmt.Errorf("create fixed income holding: %w", err)
	}
	return h, nil
}

// ListFixedIncomeHoldingsByUserID returns the caller's fixed-income holdings (oldest first).
func (r Repository) ListFixedIncomeHoldingsByUserID(ctx context.Context, userID string) ([]portfolio.FixedIncomeHolding, error) {
	const q = `
		SELECT id::text, user_id::text, name, institution, invested_amount_centavos,
			annual_rate_bps, maturity_date, liquidity_type, created_at, updated_at
		FROM fixed_income_holdings WHERE user_id = $1::uuid ORDER BY created_at`
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list fixed income holdings: %w", err)
	}
	defer rows.Close()

	var out []portfolio.FixedIncomeHolding
	for rows.Next() {
		var (
			id, uid, name, institution, liquidity string
			amount                                int64
			rate                                  int
			maturity                              sql.NullTime
			created, updated                      time.Time
		)
		if err := rows.Scan(&id, &uid, &name, &institution, &amount, &rate, &maturity, &liquidity, &created, &updated); err != nil {
			return nil, fmt.Errorf("list fixed income holdings: %w", err)
		}
		h, err := rebuildFixedIncome(id, uid, name, institution, amount, rate, maturity, liquidity, created, updated)
		if err != nil {
			return nil, fmt.Errorf("list fixed income holdings: %w", err)
		}
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list fixed income holdings: %w", err)
	}
	return out, nil
}

// UpdateFixedIncomeHolding replaces the mutable fields of a holding the caller owns. A
// missing or unowned row → ErrHoldingNotFound.
func (r Repository) UpdateFixedIncomeHolding(ctx context.Context, h portfolio.FixedIncomeHolding) (portfolio.FixedIncomeHolding, error) {
	const q = `
		UPDATE fixed_income_holdings
		SET name = $3, institution = $4, invested_amount_centavos = $5, annual_rate_bps = $6,
			maturity_date = $7, liquidity_type = $8, updated_at = $9
		WHERE id = $1::uuid AND user_id = $2::uuid
		RETURNING created_at, updated_at`
	err := r.db.QueryRowContext(ctx, q,
		h.ID, h.UserID, h.Name, h.Institution, h.InvestedAmountCentavos, h.AnnualRateBps,
		nullableDate(h.MaturityDate), string(h.LiquidityType), h.UpdatedAt).
		Scan(&h.CreatedAt, &h.UpdatedAt)
	if notFound(err) {
		return portfolio.FixedIncomeHolding{}, portfolio.ErrHoldingNotFound
	}
	if err != nil {
		return portfolio.FixedIncomeHolding{}, fmt.Errorf("update fixed income holding: %w", err)
	}
	return h, nil
}

// DeleteFixedIncomeHolding removes a holding the caller owns. A missing or unowned row →
// ErrHoldingNotFound.
func (r Repository) DeleteFixedIncomeHolding(ctx context.Context, userID, id string) error {
	return r.deleteHolding(ctx, "fixed_income_holdings", userID, id)
}

// --- helpers ---

// deleteHolding deletes a row from table scoped by (id, user_id). table is a trusted
// constant (never user input), so interpolating it is safe.
func (r Repository) deleteHolding(ctx context.Context, table, userID, id string) error {
	q := fmt.Sprintf(`DELETE FROM %s WHERE id = $1::uuid AND user_id = $2::uuid`, table)
	res, err := r.db.ExecContext(ctx, q, id, userID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentation {
			return portfolio.ErrHoldingNotFound // a malformed id matches no row
		}
		return fmt.Errorf("delete holding: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete holding: %w", err)
	}
	if n == 0 {
		return portfolio.ErrHoldingNotFound
	}
	return nil
}

// notFound reports whether a scoped single-row query matched nothing: either no rows, or a
// malformed UUID that can never match.
func notFound(err error) bool {
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentation
}

func rebuildFII(id, userID, ticker string, quantity int, avgPrice int64, created, updated time.Time) (portfolio.FIIHolding, error) {
	t, err := marketdata.ParseTicker(ticker)
	if err != nil {
		return portfolio.FIIHolding{}, err
	}
	q, err := portfolio.ParseQuantity(quantity)
	if err != nil {
		return portfolio.FIIHolding{}, err
	}
	return portfolio.FIIHolding{
		ID: id, UserID: userID, Ticker: t, Quantity: q,
		AveragePriceCentavos: avgPrice, CreatedAt: created, UpdatedAt: updated,
	}, nil
}

func rebuildFixedIncome(id, userID, name, institution string, amount int64, rate int, maturity sql.NullTime, liquidity string, created, updated time.Time) (portfolio.FixedIncomeHolding, error) {
	lt, err := portfolio.ParseLiquidityType(liquidity)
	if err != nil {
		return portfolio.FixedIncomeHolding{}, err
	}
	h := portfolio.FixedIncomeHolding{
		ID: id, UserID: userID, Name: name, Institution: institution,
		InvestedAmountCentavos: amount, AnnualRateBps: rate, LiquidityType: lt,
		CreatedAt: created, UpdatedAt: updated,
	}
	if maturity.Valid {
		d := maturity.Time
		h.MaturityDate = &d
	}
	return h, nil
}

func nullableDate(d *time.Time) sql.NullTime {
	if d == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *d, Valid: true}
}

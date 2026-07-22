package portfolio

import (
	"context"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
)

// Repository persists holdings, per-user scoped and ownership-checked (SPEC-102 FR-1026,
// BR-1021). Creates return the stored row (with DB-assigned id/timestamps); reads are scoped
// by user_id; updates and deletes are double-scoped by (id, user_id) and return
// ErrHoldingNotFound when no owned row matches. The userID always comes from the
// authenticated context — the repository never derives identity.
type Repository interface {
	CreateFIIHolding(ctx context.Context, h FIIHolding) (FIIHolding, error)
	ListFIIHoldingsByUserID(ctx context.Context, userID string) ([]FIIHolding, error)
	UpdateFIIHolding(ctx context.Context, h FIIHolding) (FIIHolding, error) // scoped by (id, user_id); ErrHoldingNotFound
	DeleteFIIHolding(ctx context.Context, userID, id string) error          // scoped; ErrHoldingNotFound

	CreateFixedIncomeHolding(ctx context.Context, h FixedIncomeHolding) (FixedIncomeHolding, error)
	ListFixedIncomeHoldingsByUserID(ctx context.Context, userID string) ([]FixedIncomeHolding, error)
	UpdateFixedIncomeHolding(ctx context.Context, h FixedIncomeHolding) (FixedIncomeHolding, error) // scoped; ErrHoldingNotFound
	DeleteFixedIncomeHolding(ctx context.Context, userID, id string) error                          // scoped; ErrHoldingNotFound
}

// Reader is the consumer-facing read port (SPEC-102 FR-1025): the seam the dashboard
// (SPEC-103), Fact Builder (SPEC-104), and projections (SPEC-107) read a user's holdings
// through, without coupling to HTTP or SQL. The Service satisfies it.
type Reader interface {
	ListHoldings(ctx context.Context, userID string) (Holdings, error)
}

// MacroReader supplies the latest macro indicator (SPEC-006), used to resolve a fixed-income
// holding's effective annual rate (SPEC-109 FR-1092). A missing indicator degrades gracefully
// (BR-1094) — the Service treats GetLatestMacroIndicator's error as "unavailable", never fatal.
type MacroReader interface {
	GetLatestMacroIndicator(ctx context.Context, ind marketdata.Indicator) (marketdata.MacroIndicator, error)
}

// SystemReader is a system-scoped read, deliberately separate from Repository/Reader (SPEC-007
// FR-071, BR-071): it is made on behalf of an internal worker, not an authenticated user, so the
// identity-from-context / WHERE user_id = $1 rule does not apply — there is no request user to
// scope to. It returns only public B3 tickers, never user-identifying data, and is reachable only
// from the market-data ingestion edge, never from an HTTP handler.
type SystemReader interface {
	// DistinctFIITickers returns the distinct FII tickers held across ALL users, as raw strings —
	// parsing into marketdata.Ticker happens at the ingestion edge (SPEC-007 FR-071 AC3), keeping
	// this package free of marketdata's ticker semantics.
	DistinctFIITickers(ctx context.Context) ([]string, error)
}

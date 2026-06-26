package portfolio

import (
	"errors"
	"time"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
)

// Domain sentinels — check with errors.Is.
var (
	// ErrHoldingNotFound is returned when a holding does not exist or is not owned by the
	// caller (the two are indistinguishable on purpose — no cross-user existence oracle,
	// SPEC-102 BR-1021).
	ErrHoldingNotFound = errors.New("holding not found")
	// ErrEmptyField marks a required text field (name/institution) left blank.
	ErrEmptyField = errors.New("required field is empty")
	// ErrInvalidAmount marks a non-positive money amount where a positive one is required.
	ErrInvalidAmount = errors.New("amount must be positive")
	// ErrNegativeAmount marks a negative money amount where a non-negative one is required.
	ErrNegativeAmount = errors.New("amount must not be negative")
	// ErrInvalidRate marks a negative annual rate.
	ErrInvalidRate = errors.New("rate must not be negative")
	// ErrPastMaturity marks a new at-maturity holding whose maturity date is already past.
	ErrPastMaturity = errors.New("maturity date is in the past")
	// ErrMaturityRequired marks an at-maturity holding missing its maturity date.
	ErrMaturityRequired = errors.New("maturity date is required for an at-maturity holding")
)

// FIIHolding is a registered FII position (SPEC-102 FR-001). It stores the user-entered
// cost basis (AveragePriceCentavos) — never a market-derived value (BR-1024). Money is
// int64 centavos (BR-1022). The Ticker is the shared marketdata value object (D1).
type FIIHolding struct {
	ID                   string
	UserID               string
	Ticker               marketdata.Ticker
	Quantity             Quantity
	AveragePriceCentavos int64
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// FixedIncomeHolding is a registered fixed-income position (SPEC-102 FR-002). InvestedAmount
// is the cost basis in centavos; AnnualRate is in basis points; MaturityDate is nil for a
// daily-liquidity holding and set for an at-maturity one.
type FixedIncomeHolding struct {
	ID                     string
	UserID                 string
	Name                   string
	Institution            string
	InvestedAmountCentavos int64
	AnnualRateBps          int
	MaturityDate           *time.Time
	LiquidityType          LiquidityType
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// Holdings is the aggregate the Reader returns — the caller's full set of holdings, for the
// dashboard / Fact Builder / projections to compute over (SPEC-102 FR-1025).
type Holdings struct {
	FII         []FIIHolding
	FixedIncome []FixedIncomeHolding
}

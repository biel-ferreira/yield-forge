package portfolio

import (
	"errors"
	"time"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/platform/money"
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
// daily-liquidity holding and set for an at-maturity one. IndexerType determines how
// AnnualRateBps is interpreted (SPEC-109) — see ResolveEffectiveRate. EffectiveAnnualRateBps is
// a computed, NEVER-PERSISTED field: it starts zero-value and is populated by the Service (via
// ResolveEffectiveRate) on every read/write path before the holding reaches its caller — the
// repository never selects or writes it (FR-1092/BR-1092).
//
// SPEC-110 reinterprets InvestedAmountCentavos: it is the current principal/balance basis used
// for accrual, growing via both contributions AND confirmed interest — no longer the cost basis
// by itself. TotalContributedCentavos is the new cost basis (lifetime money the user put in,
// untouched by interest, BR-1101). LastReconciledAt is the accrual clock, replacing CreatedAt for
// interest-accrual purposes. EstimatedInterestCentavos/ReconciliationDue are computed, never
// persisted — same treatment as EffectiveAnnualRateBps, populated by the Service.
type FixedIncomeHolding struct {
	ID                        string
	UserID                    string
	Name                      string
	Institution               string
	InvestedAmountCentavos    int64
	TotalContributedCentavos  int64 // new (SPEC-110): lifetime contributions, the cost basis for growth
	AnnualRateBps             int
	IndexerType               Indexer
	EffectiveAnnualRateBps    int   // computed, never persisted — see doc comment above
	EstimatedInterestCentavos int64 // computed, never persisted (SPEC-110 FR-1103) — see EstimateInterest
	ReconciliationDue         bool  // computed, never persisted (SPEC-110 FR-1105) — see IsReconciliationDue
	MaturityDate              *time.Time
	LiquidityType             LiquidityType
	LastReconciledAt          time.Time // new (SPEC-110): the accrual clock, replaces CreatedAt for accrual
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

// EstimateInterest returns the simple-interest estimate accrued since LastReconciledAt, as of
// now (SPEC-110 FR-1103) — the pre-fill hint for reconciliation. Pure — no I/O; reuses the same
// money.AccrueSimpleInterest formula the Dashboard uses, keyed off LastReconciledAt instead of
// CreatedAt.
func (h FixedIncomeHolding) EstimateInterest(now time.Time) int64 {
	return money.AccrueSimpleInterest(h.InvestedAmountCentavos, h.EffectiveAnnualRateBps, wholeDaysBetween(h.LastReconciledAt, now))
}

// IsReconciliationDue reports whether LastReconciledAt's calendar month (UTC) is strictly before
// now's calendar month (SPEC-110 FR-1105) — re-triggers automatically at the start of every new
// month with no stored/cron state. Pure — no I/O.
func (h FixedIncomeHolding) IsReconciliationDue(now time.Time) bool {
	ly, lm, _ := h.LastReconciledAt.UTC().Date()
	ny, nm, _ := now.UTC().Date()
	return ly < ny || (ly == ny && lm < nm)
}

// wholeDaysBetween returns the whole days elapsed from `from` to `to`, floored at 0 — mirrors
// internal/dashboard's daysBetween exactly; duplicated rather than exported/imported to keep
// portfolio from depending on dashboard (the dependency runs the other way, SPEC-103 D1).
func wholeDaysBetween(from, to time.Time) int {
	d := to.Sub(from)
	if d < 0 {
		return 0
	}
	return int(d / (24 * time.Hour))
}

// ResolveEffectiveRate resolves the holding's current effective annual rate from its stored
// AnnualRateBps + IndexerType and the latest macro readings (SPEC-109 FR-1092). Pure — no I/O;
// the caller fetches macro (keyed by Indicator) via its own MacroReader port before calling
// this. Never errors: an Indexer other than Prefixado whose reference indicator is absent from
// macro degrades to the raw stored value, unresolved (BR-1094/PLAN-109 D3) — a transient,
// self-healing gap (e.g. before the first ingestion run), never a crash or a silent zero.
func (h FixedIncomeHolding) ResolveEffectiveRate(macro map[marketdata.Indicator]marketdata.MacroIndicator) int {
	switch h.IndexerType {
	case IndexerCDIPercentual:
		if cdi, ok := macro[marketdata.IndicatorCDI]; ok {
			return int(money.ApplyBps(cdi.Value, h.AnnualRateBps))
		}
	case IndexerIPCASpread:
		if ipca, ok := macro[marketdata.IndicatorIPCA]; ok {
			return h.AnnualRateBps + int(ipca.Value)
		}
	}
	// Prefixado, an unrecognized/zero-value indexer, or a missing macro reading: pass through.
	return h.AnnualRateBps
}

// Holdings is the aggregate the Reader returns — the caller's full set of holdings, for the
// dashboard / Fact Builder / projections to compute over (SPEC-102 FR-1025).
type Holdings struct {
	FII         []FIIHolding
	FixedIncome []FixedIncomeHolding
}

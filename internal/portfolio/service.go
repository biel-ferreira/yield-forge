package portfolio

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/platform/clock"
)

// Service is the portfolio application logic (SPEC-102 FR-1022/FR-1023). It depends only on
// the Repository port and the Clock, so it is pure and unit-testable with hand-written
// fakes. It satisfies Reader, the consumer port the dashboard/facts/projections read through.
// The userID is always supplied by the caller from the authenticated context (BR-1021).
type Service struct {
	repo  Repository
	clock clock.Clock
}

var _ Reader = (*Service)(nil)

// NewService builds a Service over the repository and clock.
func NewService(repo Repository, clk clock.Clock) *Service {
	return &Service{repo: repo, clock: clk}
}

// FIIInput is the raw, edge-validated input for creating/updating an FII holding. Money is
// integer centavos (BR-1022); identity is never part of it (BR-1021).
type FIIInput struct {
	Ticker               string
	Quantity             int
	AveragePriceCentavos int64
}

// FixedIncomeInput is the raw input for creating/updating a fixed-income holding.
type FixedIncomeInput struct {
	Name                   string
	Institution            string
	InvestedAmountCentavos int64
	AnnualRateBps          int
	MaturityDate           *time.Time
	LiquidityType          string
}

// --- FII holdings ---

// CreateFIIHolding validates the input and inserts an FII holding for userID.
func (s *Service) CreateFIIHolding(ctx context.Context, userID string, in FIIInput) (FIIHolding, error) {
	h, err := s.buildFIIHolding(in)
	if err != nil {
		return FIIHolding{}, err
	}
	now := s.clock.Now()
	h.UserID, h.CreatedAt, h.UpdatedAt = userID, now, now
	return s.repo.CreateFIIHolding(ctx, h)
}

// ListFIIHoldings returns the caller's FII holdings.
func (s *Service) ListFIIHoldings(ctx context.Context, userID string) ([]FIIHolding, error) {
	return s.repo.ListFIIHoldingsByUserID(ctx, userID)
}

// UpdateFIIHolding validates the input and replaces the caller's holding id. A holding the
// caller does not own yields ErrHoldingNotFound (from the repository's scoped update).
func (s *Service) UpdateFIIHolding(ctx context.Context, userID, id string, in FIIInput) (FIIHolding, error) {
	h, err := s.buildFIIHolding(in)
	if err != nil {
		return FIIHolding{}, err
	}
	h.ID, h.UserID, h.UpdatedAt = id, userID, s.clock.Now()
	return s.repo.UpdateFIIHolding(ctx, h)
}

// DeleteFIIHolding removes the caller's holding id (ErrHoldingNotFound if absent/unowned).
func (s *Service) DeleteFIIHolding(ctx context.Context, userID, id string) error {
	return s.repo.DeleteFIIHolding(ctx, userID, id)
}

func (s *Service) buildFIIHolding(in FIIInput) (FIIHolding, error) {
	ticker, err := marketdata.ParseTicker(in.Ticker)
	if err != nil {
		return FIIHolding{}, err
	}
	quantity, err := ParseQuantity(in.Quantity)
	if err != nil {
		return FIIHolding{}, err
	}
	if in.AveragePriceCentavos < 0 {
		return FIIHolding{}, fmt.Errorf("build fii holding: %w", ErrNegativeAmount)
	}
	return FIIHolding{Ticker: ticker, Quantity: quantity, AveragePriceCentavos: in.AveragePriceCentavos}, nil
}

// --- Fixed income holdings ---

// CreateFixedIncomeHolding validates the input and inserts a fixed-income holding. The
// at-maturity past-date rule is enforced here (PRD Epic 1, BR-1023) using the Clock.
func (s *Service) CreateFixedIncomeHolding(ctx context.Context, userID string, in FixedIncomeInput) (FixedIncomeHolding, error) {
	h, err := s.buildFixedIncomeHolding(in, true)
	if err != nil {
		return FixedIncomeHolding{}, err
	}
	now := s.clock.Now()
	h.UserID, h.CreatedAt, h.UpdatedAt = userID, now, now
	return s.repo.CreateFixedIncomeHolding(ctx, h)
}

// ListFixedIncomeHoldings returns the caller's fixed-income holdings.
func (s *Service) ListFixedIncomeHoldings(ctx context.Context, userID string) ([]FixedIncomeHolding, error) {
	return s.repo.ListFixedIncomeHoldingsByUserID(ctx, userID)
}

// UpdateFixedIncomeHolding validates and replaces the caller's holding id. The past-maturity
// rule applies to creation only (PRD: "for new FI"), so an existing holding can be edited
// even as its maturity nears.
func (s *Service) UpdateFixedIncomeHolding(ctx context.Context, userID, id string, in FixedIncomeInput) (FixedIncomeHolding, error) {
	h, err := s.buildFixedIncomeHolding(in, false)
	if err != nil {
		return FixedIncomeHolding{}, err
	}
	h.ID, h.UserID, h.UpdatedAt = id, userID, s.clock.Now()
	return s.repo.UpdateFixedIncomeHolding(ctx, h)
}

// DeleteFixedIncomeHolding removes the caller's holding id (ErrHoldingNotFound if absent/unowned).
func (s *Service) DeleteFixedIncomeHolding(ctx context.Context, userID, id string) error {
	return s.repo.DeleteFixedIncomeHolding(ctx, userID, id)
}

func (s *Service) buildFixedIncomeHolding(in FixedIncomeInput, isCreate bool) (FixedIncomeHolding, error) {
	name := strings.TrimSpace(in.Name)
	institution := strings.TrimSpace(in.Institution)
	if name == "" || institution == "" {
		return FixedIncomeHolding{}, fmt.Errorf("build fixed income holding: %w", ErrEmptyField)
	}
	if in.InvestedAmountCentavos <= 0 {
		return FixedIncomeHolding{}, fmt.Errorf("build fixed income holding: %w", ErrInvalidAmount)
	}
	if in.AnnualRateBps < 0 {
		return FixedIncomeHolding{}, fmt.Errorf("build fixed income holding: %w", ErrInvalidRate)
	}
	lt, err := ParseLiquidityType(in.LiquidityType)
	if err != nil {
		return FixedIncomeHolding{}, err
	}

	h := FixedIncomeHolding{
		Name: name, Institution: institution,
		InvestedAmountCentavos: in.InvestedAmountCentavos, AnnualRateBps: in.AnnualRateBps,
		LiquidityType: lt,
	}

	if lt.RequiresMaturity() {
		if in.MaturityDate == nil {
			return FixedIncomeHolding{}, fmt.Errorf("build fixed income holding: %w", ErrMaturityRequired)
		}
		maturity := in.MaturityDate.UTC()
		if isCreate && maturity.Before(s.today()) {
			return FixedIncomeHolding{}, fmt.Errorf("build fixed income holding: %w", ErrPastMaturity)
		}
		h.MaturityDate = &maturity
	}
	// A daily-liquidity holding never carries a maturity date (normalized to nil).
	return h, nil
}

// today is the current date (midnight UTC) per the Clock, for the past-maturity comparison.
func (s *Service) today() time.Time {
	n := s.clock.Now().UTC()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}

// --- Reader ---

// ListHoldings returns the caller's full set of holdings (SPEC-102 FR-1025).
func (s *Service) ListHoldings(ctx context.Context, userID string) (Holdings, error) {
	fii, err := s.repo.ListFIIHoldingsByUserID(ctx, userID)
	if err != nil {
		return Holdings{}, fmt.Errorf("list holdings: %w", err)
	}
	fixedIncome, err := s.repo.ListFixedIncomeHoldingsByUserID(ctx, userID)
	if err != nil {
		return Holdings{}, fmt.Errorf("list holdings: %w", err)
	}
	return Holdings{FII: fii, FixedIncome: fixedIncome}, nil
}

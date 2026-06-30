package health

import (
	"context"
	"errors"
	"fmt"

	"github.com/biel-ferreira/yield-forge/internal/dashboard"
	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/platform/money"
	"github.com/biel-ferreira/yield-forge/internal/portfolio"
	"github.com/biel-ferreira/yield-forge/internal/profile"
)

// Service computes the Health Score for a user (SPEC-106). It composes the dashboard/profile/
// holdings/macro reads into the flat Inputs and runs the deterministic Compute; the score is never
// LLM-touched (the narrative is layered on in Phase 3). Identity comes from the caller's context.
type Service struct {
	dashboards DashboardReader
	profiles   ProfileReader
	holdings   HoldingsReader
	macro      MacroReader
}

// NewService builds the health service over the read seams.
func NewService(d DashboardReader, p ProfileReader, h HoldingsReader, m MacroReader) *Service {
	return &Service{dashboards: d, profiles: p, holdings: h, macro: m}
}

// Score builds the inputs and computes the reproducible Health Score for userID.
func (s *Service) Score(ctx context.Context, userID string) (HealthScore, error) {
	in, err := s.buildInputs(ctx, userID)
	if err != nil {
		return HealthScore{}, err
	}
	return Compute(in), nil
}

// buildInputs composes the structured Inputs from the read seams. A not-set profile and a missing
// macro degrade gracefully (the affected factors handle it); a dashboard/holdings error surfaces.
func (s *Service) buildInputs(ctx context.Context, userID string) (Inputs, error) {
	d, err := s.dashboards.GetDashboard(ctx, userID)
	if err != nil {
		return Inputs{}, fmt.Errorf("health inputs: dashboard: %w", err)
	}
	held, err := s.holdings.ListHoldings(ctx, userID)
	if err != nil {
		return Inputs{}, fmt.Errorf("health inputs: holdings: %w", err)
	}

	in := dashboardInputs(d)
	in.HoldingsCount = len(held.FII) + len(held.FixedIncome)
	in.LiquidValueCentavos = liquidValue(in.FIIValueCentavos, in.FixedIncomeValueCentavos, held.FixedIncome)

	if err := s.addProfile(ctx, userID, &in); err != nil {
		return Inputs{}, err
	}
	if err := s.addMacro(ctx, &in); err != nil {
		return Inputs{}, err
	}
	return in, nil
}

func (s *Service) addProfile(ctx context.Context, userID string, in *Inputs) error {
	p, err := s.profiles.GetProfile(ctx, userID)
	if errors.Is(err, profile.ErrProfileNotFound) {
		return nil // goal-alignment + risk-exposure are omitted (renormalised) — not an error
	}
	if err != nil {
		return fmt.Errorf("health inputs: profile: %w", err)
	}
	in.Risk = p.Risk
	in.HasProfile = true
	return nil
}

func (s *Service) addMacro(ctx context.Context, in *Inputs) error {
	m, err := s.macro.GetLatestMacroIndicator(ctx, marketdata.IndicatorSELIC)
	if errors.Is(err, marketdata.ErrMacroNotFound) {
		return nil // neutral market tilt — not an error
	}
	if err != nil {
		return fmt.Errorf("health inputs: macro: %w", err)
	}
	in.SelicBps = int(m.Value)
	in.HasMacro = true
	return nil
}

// dashboardInputs extracts the allocation/concentration facts from the computed dashboard.
func dashboardInputs(d dashboard.Dashboard) Inputs {
	in := Inputs{CurrentValueCentavos: d.Summary.CurrentValueCentavos}
	for _, c := range d.Allocation {
		switch c.Class {
		case dashboard.ClassFII:
			in.FIIValueCentavos = c.ValueCentavos
		case dashboard.ClassFixedIncome:
			in.FixedIncomeValueCentavos = c.ValueCentavos
		}
		if c.ShareBps > in.LargestClassBps {
			in.LargestClassBps = c.ShareBps
		}
	}
	in.FIISectorCount = len(d.FIISectors)
	for _, sec := range d.FIISectors {
		if sec.ShareBps > in.LargestSectorBps {
			in.LargestSectorBps = sec.ShareBps
		}
	}
	return in
}

// liquidValue is the readily-liquid patrimony: all FIIs plus the daily-liquidity share of the
// fixed income (split by invested amount, applied to the current FI value). SPEC-106 FR-1064.
func liquidValue(fiiValue, fiValue int64, fi []portfolio.FixedIncomeHolding) int64 {
	var totalInvested, dailyInvested int64
	for _, h := range fi {
		totalInvested += h.InvestedAmountCentavos
		if h.LiquidityType == portfolio.LiquidityDaily {
			dailyInvested += h.InvestedAmountCentavos
		}
	}
	liquidFI := int64(0)
	if totalInvested > 0 {
		liquidFI = money.ApplyBps(fiValue, money.ShareBps(dailyInvested, totalInvested))
	}
	return fiiValue + liquidFI
}

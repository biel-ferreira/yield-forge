package projection

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/trace"

	"github.com/biel-ferreira/yield-forge/internal/platform/money"
	"github.com/biel-ferreira/yield-forge/internal/platform/observability"
	"github.com/biel-ferreira/yield-forge/internal/portfolio"
)

// Service computes the projections for a user (SPEC-107). It composes the dashboard (current value
// + FII income) and holdings (fixed-income income) reads into the flat Inputs and runs the pure
// Compute. Identity comes from the caller's context; the contribution + horizon are request inputs.
type Service struct {
	dashboards DashboardReader
	holdings   HoldingsReader
	tracer     trace.Tracer
}

// NewService builds the projection service over the read seams.
func NewService(d DashboardReader, h HoldingsReader) *Service {
	return &Service{dashboards: d, holdings: h, tracer: observability.Tracer("projection")}
}

// Project builds the inputs and computes the projections for userID over the given monthly
// contribution and horizon (both already validated at the edge).
func (s *Service) Project(ctx context.Context, userID string, monthlyContributionCentavos int64, horizonYears int) (Projections, error) {
	// Span over the read + compute for latency visibility; it carries NO content — no figures,
	// holdings, or series reach telemetry (BR-505/FR-1076).
	ctx, span := s.tracer.Start(ctx, "projection.compute")
	defer span.End()

	d, err := s.dashboards.GetDashboard(ctx, userID)
	if err != nil {
		return Projections{}, fmt.Errorf("project: dashboard: %w", err)
	}
	held, err := s.holdings.ListHoldings(ctx, userID)
	if err != nil {
		return Projections{}, fmt.Errorf("project: holdings: %w", err)
	}

	return Compute(Inputs{
		CurrentValueCentavos:        d.Summary.CurrentValueCentavos,
		FIIAnnualIncomeCentavos:     d.Summary.MonthlyIncomeCentavos * 12, // dashboard FII monthly → annual
		FIAnnualIncomeCentavos:      fixedIncomeAnnual(held.FixedIncome),
		MonthlyContributionCentavos: monthlyContributionCentavos,
		HorizonYears:                horizonYears,
	}), nil
}

// fixedIncomeAnnual sums the annual income of the fixed-income holdings: Σ invested × annual
// rate, using each holding's resolved EffectiveAnnualRateBps (SPEC-109) rather than the raw
// stored rate — for a prefixado holding they're equal; cdi_percentual/ipca_spread use the
// current resolved rate.
func fixedIncomeAnnual(fi []portfolio.FixedIncomeHolding) int64 {
	var total int64
	for _, h := range fi {
		total += money.ApplyBps(h.InvestedAmountCentavos, h.EffectiveAnnualRateBps)
	}
	return total
}

package projection

import (
	"context"

	"github.com/biel-ferreira/yield-forge/internal/insight"
)

// BuildProjectionFacts returns the DETERMINISTIC projection as grounding facts (income + net-worth
// scenarios) — the seam the Conversational Copilot (SPEC-108) uses to ground a "daqui a N anos" /
// passive-income chat turn. The projection is already LLM-free, so this never invents numbers; the
// facts are integers only. A missing horizon uses the caller's default (validated at the edge).
func (s *Service) BuildProjectionFacts(ctx context.Context, userID string, monthlyContributionCentavos int64, horizonYears int) (insight.Facts, error) {
	ps, err := s.Project(ctx, userID, monthlyContributionCentavos, horizonYears)
	if err != nil {
		return nil, err
	}

	income := make([]map[string]any, 0, len(ps.Income))
	for _, i := range ps.Income {
		income = append(income, map[string]any{
			"scenario":         string(i.Scenario),
			"monthly_centavos": i.MonthlyCentavos,
			"annual_centavos":  i.AnnualCentavos,
		})
	}
	netWorth := make([]map[string]any, 0, len(ps.NetWorth))
	for _, n := range ps.NetWorth {
		final := int64(0)
		if len(n.Points) > 0 {
			final = n.Points[len(n.Points)-1].ValueCentavos
		}
		netWorth = append(netWorth, map[string]any{
			"scenario":                      string(n.Scenario),
			"horizon_years":                 n.Assumptions.HorizonYears,
			"monthly_contribution_centavos": n.Assumptions.MonthlyContributionCentavos,
			"final_value_centavos":          final,
		})
	}
	return insight.Facts{
		"income_scenarios":    income,
		"net_worth_scenarios": netWorth,
	}, nil
}

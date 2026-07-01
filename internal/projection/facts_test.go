package projection

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/dashboard"
	"github.com/biel-ferreira/yield-forge/internal/portfolio"
)

func TestBuildProjectionFacts(t *testing.T) {
	svc := NewService(
		fakeDashboard{d: dashboard.Dashboard{Summary: dashboard.Summary{CurrentValueCentavos: 100_000, MonthlyIncomeCentavos: 500}}},
		fakeHoldings{h: portfolio.Holdings{}},
	)

	facts, err := svc.BuildProjectionFacts(context.Background(), "u1", 10_000, 10)
	require.NoError(t, err)

	income, ok := facts["income_scenarios"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, income, 3, "pessimistic / base / optimistic")

	netWorth, ok := facts["net_worth_scenarios"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, netWorth, 3)
	require.Equal(t, 10, netWorth[0]["horizon_years"])
	require.Contains(t, netWorth[0], "final_value_centavos")
}

package projection

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/dashboard"
	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/portfolio"
)

type fakeDashboard struct {
	d   dashboard.Dashboard
	err error
}

func (f fakeDashboard) GetDashboard(context.Context, string) (dashboard.Dashboard, error) {
	return f.d, f.err
}

type fakeHoldings struct {
	h   portfolio.Holdings
	err error
}

func (f fakeHoldings) ListHoldings(context.Context, string) (portfolio.Holdings, error) {
	return f.h, f.err
}

func TestService_Project_ComposesInputs(t *testing.T) {
	d := dashboard.Dashboard{Summary: dashboard.Summary{CurrentValueCentavos: 2_820_000, MonthlyIncomeCentavos: 11_000}}
	held := portfolio.Holdings{
		FII: []portfolio.FIIHolding{{Ticker: marketdata.MustParseTicker("HGLG11")}},
		FixedIncome: []portfolio.FixedIncomeHolding{
			// EffectiveAnnualRateBps (SPEC-109) is what fixedIncomeAnnual reads; ListHoldings
			// resolves it in production — set directly here for this pure-function test.
			{InvestedAmountCentavos: 1_000_000, AnnualRateBps: 1_200, EffectiveAnnualRateBps: 1_200}, // 12% of 1M = 120_000/yr
		},
	}
	svc := NewService(fakeDashboard{d: d}, fakeHoldings{h: held})

	ps, err := svc.Project(context.Background(), "u1", 50_000, 10)
	require.NoError(t, err)

	// FII annual = 11_000 × 12 = 132_000; FI annual = 12% × 1_000_000 = 120_000; base = 252_000.
	require.Equal(t, int64(252_000), incomeOf(ps, ScenarioBase).AnnualCentavos)
	// Net worth starts at the current value and spans year 0..10.
	base := netWorthOf(ps, ScenarioBase)
	require.Equal(t, int64(2_820_000), base.Points[0].ValueCentavos)
	require.Len(t, base.Points, 11)
	require.Equal(t, int64(50_000), base.Assumptions.MonthlyContributionCentavos)
}

func TestService_Project_Deterministic(t *testing.T) {
	svc := NewService(
		fakeDashboard{d: dashboard.Dashboard{Summary: dashboard.Summary{CurrentValueCentavos: 100_000, MonthlyIncomeCentavos: 500}}},
		fakeHoldings{h: portfolio.Holdings{}},
	)
	a, err := svc.Project(context.Background(), "u1", 10_000, 5)
	require.NoError(t, err)
	b, err := svc.Project(context.Background(), "u1", 10_000, 5)
	require.NoError(t, err)
	require.Equal(t, a, b, "same inputs → identical projection")
}

func TestFixedIncomeAnnual(t *testing.T) {
	require.Equal(t, int64(0), fixedIncomeAnnual(nil))
	require.Equal(t, int64(120_000), fixedIncomeAnnual([]portfolio.FixedIncomeHolding{
		{InvestedAmountCentavos: 1_000_000, AnnualRateBps: 1_200, EffectiveAnnualRateBps: 1_200},
	}))
}

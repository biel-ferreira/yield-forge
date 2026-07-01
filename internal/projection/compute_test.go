package projection

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func incomeOf(ps Projections, sc Scenario) ScenarioIncome {
	for _, i := range ps.Income {
		if i.Scenario == sc {
			return i
		}
	}
	return ScenarioIncome{}
}

func netWorthOf(ps Projections, sc Scenario) ScenarioNetWorth {
	for _, n := range ps.NetWorth {
		if n.Scenario == sc {
			return n
		}
	}
	return ScenarioNetWorth{}
}

func sampleInputs() Inputs {
	// R$1000 patrimony, R$80/yr income (8% yield), R$100/mo contribution, 10-year horizon.
	return Inputs{
		CurrentValueCentavos: 100_000, FIIAnnualIncomeCentavos: 8_000, FIAnnualIncomeCentavos: 0,
		MonthlyContributionCentavos: 10_000, HorizonYears: 10,
	}
}

func TestCompute_Reproducible(t *testing.T) {
	in := sampleInputs()
	require.Equal(t, Compute(in), Compute(in), "same inputs → identical projection")
}

func TestCompute_IncomeScenarios(t *testing.T) {
	ps := Compute(sampleInputs())
	require.Len(t, ps.Income, 3)
	require.NotEmpty(t, ps.Disclaimer)

	// base reconciles exactly with the holdings/market (8000/yr); ±delta = ±2% of R$1000 = R$20.
	require.Equal(t, int64(8_000), incomeOf(ps, ScenarioBase).AnnualCentavos)
	require.Equal(t, int64(667), incomeOf(ps, ScenarioBase).MonthlyCentavos, "8000/12 half-up")
	require.Equal(t, int64(6_000), incomeOf(ps, ScenarioPessimistic).AnnualCentavos, "base − 2% of value")
	require.Equal(t, int64(10_000), incomeOf(ps, ScenarioOptimistic).AnnualCentavos, "base + 2% of value")
	require.Equal(t, -200, incomeOf(ps, ScenarioPessimistic).Assumptions.YieldAdjBps)
}

func TestCompute_NetWorthShapeAndMonotonic(t *testing.T) {
	ps := Compute(sampleInputs())
	base := netWorthOf(ps, ScenarioBase)
	require.Len(t, base.Points, 11, "year 0 through year 10")
	require.Equal(t, 0, base.Points[0].Year)
	require.Equal(t, int64(100_000), base.Points[0].ValueCentavos, "year 0 = current value")
	require.Equal(t, 10, base.Points[10].Year)
	// Positive yield + contribution → strictly increasing.
	for i := 1; i < len(base.Points); i++ {
		require.Greater(t, base.Points[i].ValueCentavos, base.Points[i-1].ValueCentavos)
	}
	require.Greater(t, base.Points[10].ValueCentavos, int64(100_000), "grows past the starting value")
}

func TestProjectNetWorth_PureContributionAccumulates(t *testing.T) {
	// Empty portfolio (no yield basis) → the BASE scenario yield is 0, so net worth is the pure
	// accumulation of contributions: year k = k × 12 × contribution. Exactly hand-computable.
	in := Inputs{MonthlyContributionCentavos: 10_000, HorizonYears: 2}
	ps := Compute(in)
	base := netWorthOf(ps, ScenarioBase)
	require.Equal(t, int64(0), base.Points[0].ValueCentavos)
	require.Equal(t, int64(120_000), base.Points[1].ValueCentavos, "12 × 10_000")
	require.Equal(t, int64(240_000), base.Points[2].ValueCentavos, "24 × 10_000")
}

func TestCompute_EmptyPortfolio(t *testing.T) {
	ps := Compute(Inputs{HorizonYears: 5})
	for _, i := range ps.Income {
		require.Equal(t, int64(0), i.AnnualCentavos, "no holdings → zero income")
		require.Equal(t, int64(0), i.MonthlyCentavos)
	}
	base := netWorthOf(ps, ScenarioBase)
	require.Len(t, base.Points, 6)
	for _, p := range base.Points {
		require.Equal(t, int64(0), p.ValueCentavos, "no value, no contribution → flat zero")
	}
}

func TestCompute_LongHorizonNoOverflow(t *testing.T) {
	// A large portfolio compounding for 40 years must stay int64-safe and remain positive.
	in := Inputs{
		CurrentValueCentavos: 100_000_000_00, FIIAnnualIncomeCentavos: 8_000_000_00,
		MonthlyContributionCentavos: 1_000_000_00, HorizonYears: 40,
	}
	ps := Compute(in)
	opt := netWorthOf(ps, ScenarioOptimistic)
	require.Len(t, opt.Points, 41)
	last := opt.Points[40].ValueCentavos
	require.Greater(t, last, in.CurrentValueCentavos)
	require.Greater(t, last, int64(0), "no overflow to negative")
}

func TestParseScenario(t *testing.T) {
	sc, err := ParseScenario("  Optimistic ")
	require.NoError(t, err)
	require.Equal(t, ScenarioOptimistic, sc)
	_, err = ParseScenario("wishful")
	require.ErrorIs(t, err, ErrUnknownScenario)
}

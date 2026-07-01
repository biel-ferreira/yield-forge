package projection

import (
	"fmt"

	"github.com/biel-ferreira/yield-forge/internal/platform/money"
)

// Inputs is the flat, structured snapshot the projection is computed from (SPEC-107 §6). Keeping it
// primitive makes Compute a pure, reproducible function — the service derives these from the
// dashboard + holdings reads (Phase 2). Money is int64 centavos, rates int bps.
type Inputs struct {
	CurrentValueCentavos        int64 // full patrimony (net worth start)
	FIIAnnualIncomeCentavos     int64 // Σ FII dividends, annualised (dashboard monthly × 12)
	FIAnnualIncomeCentavos      int64 // Σ fixed income (invested × annual rate)
	MonthlyContributionCentavos int64 // configured new money each month (≥ 0)
	HorizonYears                int   // projection horizon in years (bounded by the edge)
}

// Compute produces the deterministic income + net-worth projections (SPEC-107 FR-1071/1072). Same
// Inputs → same figures and series. Base income reconciles exactly with the holdings/market; the
// pessimistic/optimistic scenarios adjust the income yield by ±spreadBps; net worth compounds the
// scenario yield monthly (reinvested income) plus the contribution.
func Compute(in Inputs) Projections {
	baseAnnual := in.FIIAnnualIncomeCentavos + in.FIAnnualIncomeCentavos
	baseYieldBps := money.ShareBps(baseAnnual, in.CurrentValueCentavos)
	// The income change for a ±spreadBps yield move = that share of the current value.
	delta := money.ApplyBps(in.CurrentValueCentavos, spreadBps)

	income := make([]ScenarioIncome, 0, len(AllScenarios))
	netWorth := make([]ScenarioNetWorth, 0, len(AllScenarios))
	for _, sc := range AllScenarios {
		adj := sc.yieldAdjBps()

		// Scale the ±spreadBps income change by this scenario's adjustment: at adj = ±spreadBps the
		// division is exact, giving base ± delta; at adj = 0 it is base. Generalises to any adj.
		annual := baseAnnual + delta*int64(adj)/spreadBps
		if annual < 0 {
			annual = 0
		}
		income = append(income, ScenarioIncome{
			Scenario:        sc,
			AnnualCentavos:  annual,
			MonthlyCentavos: monthlyFromAnnual(annual),
			Assumptions: IncomeAssumptions{
				YieldAdjBps: adj,
				Note:        fmt.Sprintf("rendimento base ajustado em %+d bps; valores nominais.", adj),
			},
		})

		scenarioYieldBps := baseYieldBps + adj
		if scenarioYieldBps < 0 {
			scenarioYieldBps = 0
		}
		netWorth = append(netWorth, ScenarioNetWorth{
			Scenario: sc,
			Points:   projectNetWorth(in, scenarioYieldBps),
			Assumptions: NetWorthAssumptions{
				YieldAdjBps:                 adj,
				MonthlyContributionCentavos: in.MonthlyContributionCentavos,
				HorizonYears:                in.HorizonYears,
				Note:                        fmt.Sprintf("rendimento reinvestido a %d bps a.a. + aporte mensal; sem valorização de preço; nominal.", scenarioYieldBps),
			},
		})
	}
	return Projections{Income: income, NetWorth: netWorth, Disclaimer: Disclaimer}
}

// projectNetWorth compounds the value monthly at the scenario yield (reinvested income) plus the
// monthly contribution, snapshotting at each year boundary — year 0 (current value) through the
// horizon (SPEC-107 FR-1072/D4/D6). Deterministic; half-up per month; big.Int-guarded via ApplyBps.
func projectNetWorth(in Inputs, yieldBps int) []NetWorthPoint {
	monthlyRateBps := yieldBps / 12 // documented monthly compounding of the annual yield
	value := in.CurrentValueCentavos

	points := make([]NetWorthPoint, 0, in.HorizonYears+1)
	points = append(points, NetWorthPoint{Year: 0, ValueCentavos: value})

	months := in.HorizonYears * 12
	for m := 1; m <= months; m++ {
		value += money.ApplyBps(value, monthlyRateBps) + in.MonthlyContributionCentavos
		if m%12 == 0 {
			points = append(points, NetWorthPoint{Year: m / 12, ValueCentavos: value})
		}
	}
	return points
}

// monthlyFromAnnual returns annual/12 rounded half-up; non-positive input yields 0.
func monthlyFromAnnual(annual int64) int64 {
	if annual <= 0 {
		return 0
	}
	return (annual*2 + 12) / 24 // half-up: (2*annual + 12) / (2*12)
}

package projection

import (
	"errors"
	"fmt"
	"strings"
)

// ErrUnknownScenario marks an unrecognised scenario name (SPEC-107). Check with errors.Is.
var ErrUnknownScenario = errors.New("unknown scenario")

// Disclaimer labels every projection a non-guaranteed estimate (SPEC-107 FR-1074 / FR-014).
const Disclaimer = "Estas projeções são estimativas baseadas nos dados atuais e nas premissas " +
	"mostradas — não são garantias de rentabilidade futura nem recomendação de investimento."

// spreadBps is the yield adjustment (±) applied to the base scenario for the pessimistic /
// optimistic scenarios (SPEC-107 D5). Documented and tunable.
const spreadBps = 200

// Scenario is a projection scenario (SPEC-107 FR-1073). Closed enum.
type Scenario string

const (
	ScenarioPessimistic Scenario = "pessimistic"
	ScenarioBase        Scenario = "base"
	ScenarioOptimistic  Scenario = "optimistic"
)

// AllScenarios is the fixed order projections are produced and presented in.
var AllScenarios = []Scenario{ScenarioPessimistic, ScenarioBase, ScenarioOptimistic}

// ParseScenario normalises s (trim + lower-case) into a Scenario, or ErrUnknownScenario via %w.
func ParseScenario(s string) (Scenario, error) {
	switch sc := Scenario(strings.ToLower(strings.TrimSpace(s))); sc {
	case ScenarioPessimistic, ScenarioBase, ScenarioOptimistic:
		return sc, nil
	default:
		return "", fmt.Errorf("parse scenario %q: %w", s, ErrUnknownScenario)
	}
}

// yieldAdjBps is the yield adjustment (basis points) this scenario applies to the base yield.
func (sc Scenario) yieldAdjBps() int {
	switch sc {
	case ScenarioPessimistic:
		return -spreadBps
	case ScenarioOptimistic:
		return spreadBps
	default: // base
		return 0
	}
}

// IncomeAssumptions exposes the assumptions behind an income scenario (SPEC-107 BR-1073).
type IncomeAssumptions struct {
	YieldAdjBps int    // ± adjustment applied to the base income yield
	Note        string // human-readable, nominal (no inflation/tax)
}

// ScenarioIncome is the projected passive income for one scenario (SPEC-107 FR-1071).
type ScenarioIncome struct {
	Scenario        Scenario
	MonthlyCentavos int64
	AnnualCentavos  int64
	Assumptions     IncomeAssumptions
}

// NetWorthPoint is one time-series point (a year offset + the projected value) for charting.
type NetWorthPoint struct {
	Year          int
	ValueCentavos int64
}

// NetWorthAssumptions exposes the assumptions behind a net-worth scenario (SPEC-107 BR-1073).
type NetWorthAssumptions struct {
	YieldAdjBps                 int
	MonthlyContributionCentavos int64
	HorizonYears                int
	Note                        string
}

// ScenarioNetWorth is the projected net-worth series for one scenario (SPEC-107 FR-1072).
type ScenarioNetWorth struct {
	Scenario    Scenario
	Points      []NetWorthPoint
	Assumptions NetWorthAssumptions
}

// Projections is the full computed result (SPEC-107 §6). No persistence; deterministic.
type Projections struct {
	Income     []ScenarioIncome
	NetWorth   []ScenarioNetWorth
	Disclaimer string
}

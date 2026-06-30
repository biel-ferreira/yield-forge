package health

import (
	"fmt"

	"github.com/biel-ferreira/yield-forge/internal/platform/money"
	"github.com/biel-ferreira/yield-forge/internal/profile"
)

// Inputs is the flat, structured snapshot the score is computed from (SPEC-106 §6). Keeping it
// primitive makes Compute a pure, reproducible function — the service derives these from the
// dashboard/profile/holdings/macro reads (Phase 2). All money is int64 centavos, rates int bps.
type Inputs struct {
	CurrentValueCentavos     int64 // full patrimony
	FIIValueCentavos         int64
	FixedIncomeValueCentavos int64
	LiquidValueCentavos      int64 // FII + daily-liquidity fixed income
	HoldingsCount            int   // FII + fixed-income positions
	FIISectorCount           int
	LargestSectorBps         int // largest FII sector share (concentration)
	LargestClassBps          int // largest asset-class share (FII vs fixed income)

	Risk       profile.RiskProfile // "" when unset
	HasProfile bool

	SelicBps int // policy rate in basis points (1% = 100 bps); 0 when absent
	HasMacro bool
}

// Macro-tilt parameters (SPEC-106 FR-1064): as SELIC rises above a neutral band, the "healthy" mix
// tilts toward fixed income (post-fixed gets more attractive). Modest, bounded, deterministic.
const (
	selicNeutralBps = 1000 // ~10% policy rate
	maxTiltBps      = 1000 // cap the shift at 10 percentage points of the target FII share
)

// Compute produces the deterministic, reproducible Health Score (SPEC-106 FR-1061). Same Inputs →
// same score and byte-identical breakdown. An empty portfolio scores 0 (D6); a missing profile
// omits the goal/risk factors and renormalises the rest (D4).
func Compute(in Inputs) HealthScore {
	if in.CurrentValueCentavos <= 0 {
		return emptyScore()
	}

	factors := []FactorScore{
		diversification(in),
		concentration(in),
		liquidity(in),
	}
	if in.HasProfile {
		factors = append(factors, goalAlignment(in), riskExposure(in))
	}

	// Renormalise the present factors' default weights to sum exactly 10000 (reuse the allocator).
	weights := make([]int64, len(factors))
	for i, f := range factors {
		weights[i] = int64(defaultWeightBps[f.Factor])
	}
	normalised := money.AllocateBps(weights)

	values := make([]int, len(factors))
	for i := range factors {
		factors[i].WeightBps = normalised[i]
		values[i] = factors[i].Score
	}
	return HealthScore{Score: money.WeightedMeanBps(values, normalised), Factors: factors}
}

func emptyScore() HealthScore {
	const note = "Adicione posições à carteira para avaliar este fator."
	factors := make([]FactorScore, 0, len(defaultWeightBps))
	for _, f := range []Factor{FactorDiversification, FactorConcentration, FactorLiquidity, FactorGoalAlignment, FactorRiskExposure} {
		factors = append(factors, FactorScore{Factor: f, Score: 0, WeightBps: defaultWeightBps[f], Explanation: note})
	}
	return HealthScore{Score: 0, Factors: factors}
}

// diversification rewards more positions spread across more FII sectors (SPEC-106 FR-1064).
func diversification(in Inputs) FactorScore {
	holdingsPart := clamp(in.HoldingsCount*12, 0, 60) // ~5 positions saturate the holdings part
	sectorPart := clamp(in.FIISectorCount*13, 0, 40)  // ~3 sectors saturate the sector part
	return FactorScore{
		Factor: FactorDiversification,
		Score:  clamp(holdingsPart+sectorPart, 0, 100),
		Explanation: fmt.Sprintf("%d posição(ões) em %d setor(es) de FII — mais posições e setores elevam este fator.",
			in.HoldingsCount, in.FIISectorCount),
	}
}

// concentration is the inverse of the largest single class/sector share (SPEC-106 FR-1064).
func concentration(in Inputs) FactorScore {
	worst := maxInt(in.LargestClassBps, in.LargestSectorBps)
	return FactorScore{
		Factor:      FactorConcentration,
		Score:       clamp(100-worst/100, 0, 100),
		Explanation: fmt.Sprintf("maior concentração isolada: %d%% — quanto menor, mais saudável.", clamp(worst, 0, 10000)/100),
	}
}

// liquidity is the share of patrimony readily liquid (FIIs + daily-liquidity FI) (SPEC-106 FR-1064).
func liquidity(in Inputs) FactorScore {
	shareBps := money.ShareBps(in.LiquidValueCentavos, in.CurrentValueCentavos)
	return FactorScore{
		Factor:      FactorLiquidity,
		Score:       clamp(shareBps/100, 0, 100),
		Explanation: fmt.Sprintf("%d%% do patrimônio é líquido (FIIs + renda fixa de liquidez diária).", shareBps/100),
	}
}

// goalAlignment scores how close the current FII/FI mix is to the profile-implied, market-aware
// target (SPEC-106 FR-1064). Only computed when a profile is set.
func goalAlignment(in Inputs) FactorScore {
	currentFIIBps := money.ShareBps(in.FIIValueCentavos, in.FIIValueCentavos+in.FixedIncomeValueCentavos)
	targetFIIBps := targetFIIShareBps(in.Risk, in.SelicBps, in.HasMacro)
	dist := absInt(currentFIIBps - targetFIIBps)
	return FactorScore{
		Factor:      FactorGoalAlignment,
		Score:       clamp(100-dist/100, 0, 100),
		Explanation: fmt.Sprintf("FII em %d%% vs alvo %d%% (perfil + mercado) — quanto mais próximo, melhor.", currentFIIBps/100, targetFIIBps/100),
	}
}

// riskExposure penalises a portfolio carrying more equity-like risk (FII share + concentration)
// than the investor's risk tolerance supports (SPEC-106 FR-1064). Only computed with a profile.
func riskExposure(in Inputs) FactorScore {
	currentFIIBps := money.ShareBps(in.FIIValueCentavos, in.FIIValueCentavos+in.FixedIncomeValueCentavos)
	tolerance := baseFIIShareBps(in.Risk)
	overTolerance := maxInt(0, currentFIIBps-tolerance)
	concentrationLoad := maxInt(0, maxInt(in.LargestClassBps, in.LargestSectorBps)-5000) / 2
	excess := overTolerance + concentrationLoad
	return FactorScore{
		Factor:      FactorRiskExposure,
		Score:       clamp(100-excess/100, 0, 100),
		Explanation: fmt.Sprintf("exposição a risco vs tolerância do perfil (%s) — excesso reduz este fator.", in.Risk),
	}
}

// targetFIIShareBps is the market-aware target FII share: the profile base shifted toward fixed
// income as SELIC rises above the neutral band (SPEC-106 FR-1064). Deterministic and bounded.
func targetFIIShareBps(risk profile.RiskProfile, selicBps int, hasMacro bool) int {
	base := baseFIIShareBps(risk)
	tilt := 0
	if hasMacro && selicBps > selicNeutralBps {
		tilt = clamp((selicBps-selicNeutralBps)/2, 0, maxTiltBps)
	}
	return maxInt(0, base-tilt)
}

// baseFIIShareBps is the profile-implied target FII share before any market tilt (SPEC-106
// FR-1064; the same conservative→FI / aggressive→FII direction the Rebalancing Assistant uses).
func baseFIIShareBps(risk profile.RiskProfile) int {
	switch risk {
	case profile.RiskConservative:
		return 3000
	case profile.RiskAggressive:
		return 7000
	default: // moderate / unset
		return 5000
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func absInt(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

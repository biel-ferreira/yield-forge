package health

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/profile"
)

func populatedInputs() Inputs {
	return Inputs{
		CurrentValueCentavos: 100_000, FIIValueCentavos: 60_000, FixedIncomeValueCentavos: 40_000,
		LiquidValueCentavos: 80_000, HoldingsCount: 5, FIISectorCount: 3,
		LargestSectorBps: 4000, LargestClassBps: 6000,
		Risk: profile.RiskModerate, HasProfile: true, SelicBps: 1050, HasMacro: true,
	}
}

func sumWeights(factors []FactorScore) int {
	sum := 0
	for _, f := range factors {
		sum += f.WeightBps
	}
	return sum
}

func TestCompute_ShapeAndReconciliation(t *testing.T) {
	hs := Compute(populatedInputs())
	require.Len(t, hs.Factors, 5)
	require.Equal(t, 10000, sumWeights(hs.Factors), "weights reconcile to 10000")
	require.GreaterOrEqual(t, hs.Score, 0)
	require.LessOrEqual(t, hs.Score, 100)
	for _, f := range hs.Factors {
		require.GreaterOrEqual(t, f.Score, 0)
		require.LessOrEqual(t, f.Score, 100)
		require.NotEmpty(t, f.Explanation, "every factor carries an explanation (FR-1062)")
	}
}

func TestCompute_Reproducible(t *testing.T) {
	in := populatedInputs()
	// Same inputs → same score AND byte-identical breakdown (the PRD metric).
	require.Equal(t, Compute(in), Compute(in))
}

func TestCompute_EmptyPortfolio(t *testing.T) {
	hs := Compute(Inputs{})
	require.Equal(t, 0, hs.Score)
	require.Len(t, hs.Factors, 5)
	for _, f := range hs.Factors {
		require.Equal(t, 0, f.Score)
		require.NotEmpty(t, f.Explanation)
	}
}

func TestCompute_ProfileNotSetRenormalises(t *testing.T) {
	in := populatedInputs()
	in.HasProfile = false
	in.Risk = ""
	hs := Compute(in)
	require.Len(t, hs.Factors, 3, "goal-alignment + risk-exposure omitted without a profile")
	require.Equal(t, 10000, sumWeights(hs.Factors), "remaining weights renormalised to 10000")
	for _, f := range hs.Factors {
		require.NotEqual(t, FactorGoalAlignment, f.Factor)
		require.NotEqual(t, FactorRiskExposure, f.Factor)
	}
}

func TestConcentration_InverseOfLargestShare(t *testing.T) {
	// 100% one class → fully concentrated → score 0.
	require.Equal(t, 0, concentration(Inputs{LargestClassBps: 10000}).Score)
	// 45% largest → 100 - 45 = 55.
	require.Equal(t, 55, concentration(Inputs{LargestClassBps: 4500, LargestSectorBps: 3000}).Score)
}

func TestLiquidity_ShareOfPatrimony(t *testing.T) {
	require.Equal(t, 80, liquidity(Inputs{CurrentValueCentavos: 100_000, LiquidValueCentavos: 80_000}).Score)
	require.Equal(t, 100, liquidity(Inputs{CurrentValueCentavos: 100_000, LiquidValueCentavos: 100_000}).Score)
}

func TestTargetFIIShare_MarketTilt(t *testing.T) {
	// No macro → pure profile base.
	require.Equal(t, 5000, targetFIIShareBps(profile.RiskModerate, 0, false))
	require.Equal(t, 3000, targetFIIShareBps(profile.RiskConservative, 0, false))
	require.Equal(t, 7000, targetFIIShareBps(profile.RiskAggressive, 0, false))
	// SELIC just above neutral → tiny tilt toward fixed income (lower FII target).
	require.Equal(t, 4975, targetFIIShareBps(profile.RiskModerate, 1050, true), "(1050-1000)/2 = 25 bps tilt")
	// High SELIC → tilt capped at maxTiltBps.
	require.Equal(t, 4000, targetFIIShareBps(profile.RiskModerate, 5000, true), "tilt capped at 1000 bps")
}

func TestCompute_MarketAwareYetReproducible(t *testing.T) {
	// Same portfolio, different macro → the goal-alignment factor (and the score) can differ —
	// and each remains reproducible for its own inputs.
	low := populatedInputs()
	low.SelicBps, low.HasMacro = 0, false
	high := populatedInputs()
	high.SelicBps, high.HasMacro = 4000, true

	require.Equal(t, Compute(high), Compute(high), "reproducible for its own inputs")
	// The macro shifts the target, so the goal-alignment sub-score is allowed to move.
	require.NotEqual(t, goalAlignment(low).Score, goalAlignment(high).Score, "market moves the factor")
}

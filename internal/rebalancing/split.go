package rebalancing

import (
	"github.com/biel-ferreira/yield-forge/internal/dashboard"
	"github.com/biel-ferreira/yield-forge/internal/platform/money"
	"github.com/biel-ferreira/yield-forge/internal/profile"
)

// splitClasses are the asset classes the MVP allocator distributes the contribution across. Stocks
// and ETFs are out of the MVP (not ingested, not held), so the split is FII vs Fixed Income — the
// two classes the user actually holds and that the system has live data for (SPEC-105 §scope).
var splitClasses = []dashboard.AssetClass{dashboard.ClassFII, dashboard.ClassFixedIncome}

// Allocation is the current per-class portfolio value the split reasons over (SPEC-105 FR-1053a).
// It is everything the allocator needs from the dashboard facts; keeping it explicit keeps the
// allocator a pure function, independent of where the values come from.
type Allocation struct {
	TotalCentavos int64                          // current patrimony across all classes
	ByClass       map[dashboard.AssetClass]int64 // current value per class (centavos)
}

// AreaShare is one area's computed slice of the contribution (SPEC-105 FR-1053a). The basis-point
// share is the source of truth (shares sum to exactly 10 000); the centavos amount is derived.
type AreaShare struct {
	Class                   dashboard.AssetClass
	SuggestedShareBps       int
	SuggestedAmountCentavos int64
}

// targetShares is the profile-implied target mix (basis points, Σ = 10 000) the contribution is
// steered toward — a small, documented direction rule, NOT a numeric per-class target promise
// (D4/D7). Conservative tilts Fixed Income, aggressive tilts FII; unknown profiles read as moderate.
func targetShares(risk profile.RiskProfile) map[dashboard.AssetClass]int {
	switch risk {
	case profile.RiskConservative:
		return map[dashboard.AssetClass]int{dashboard.ClassFII: 3000, dashboard.ClassFixedIncome: 7000}
	case profile.RiskAggressive:
		return map[dashboard.AssetClass]int{dashboard.ClassFII: 7000, dashboard.ClassFixedIncome: 3000}
	default: // moderate / unset
		return map[dashboard.AssetClass]int{dashboard.ClassFII: 5000, dashboard.ClassFixedIncome: 5000}
	}
}

// Split computes the suggested share of the contribution per area, deterministically (SPEC-105
// FR-1053a). New money is steered toward the classes that are UNDER their profile-implied target
// after the contribution (rebalance-by-contribution); if every class is already at/above target,
// it falls back to the target mix itself. The result reconciles to exactly 10 000 bps (half-up).
// Pure: same Allocation + risk + contribution → same split, no float.
func Split(current Allocation, risk profile.RiskProfile, contribution Contribution) []AreaShare {
	targets := targetShares(risk)
	futureTotal := current.TotalCentavos + contribution.Centavos()

	// Weight each class by how far it sits BELOW its target value after the contribution.
	weights := make([]int64, len(splitClasses))
	var anyGap bool
	for i, c := range splitClasses {
		targetValue := money.ApplyBps(futureTotal, targets[c])
		if gap := targetValue - current.ByClass[c]; gap > 0 {
			weights[i] = gap
			anyGap = true
		}
	}
	// Safety net: a positive contribution normally creates proportional gaps, but if rounding
	// leaves every class at exactly zero gap, steer by the target mix itself.
	if !anyGap {
		for i, c := range splitClasses {
			weights[i] = int64(targets[c])
		}
	}

	shares := money.AllocateBps(weights)
	out := make([]AreaShare, 0, len(splitClasses))
	for i, c := range splitClasses {
		if shares[i] == 0 {
			continue
		}
		out = append(out, AreaShare{
			Class:                   c,
			SuggestedShareBps:       shares[i],
			SuggestedAmountCentavos: money.ApplyBps(contribution.Centavos(), shares[i]),
		})
	}
	return out
}

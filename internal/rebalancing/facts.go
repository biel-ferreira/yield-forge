package rebalancing

import (
	"context"

	"github.com/biel-ferreira/yield-forge/internal/dashboard"
	"github.com/biel-ferreira/yield-forge/internal/insight"
	"github.com/biel-ferreira/yield-forge/internal/marketdata"
	"github.com/biel-ferreira/yield-forge/internal/platform/money"
	"github.com/biel-ferreira/yield-forge/internal/profile"
)

// FactSource is the published SPEC-104 Fact Builder seam the engine grounds on (BR-1052). The
// rebalancing engine REUSES it rather than re-reading the dashboard/profile/macro; *engine.Service's
// FactBuilder satisfies it. Consumer-defined here (accept interfaces).
type FactSource interface {
	BuildFacts(ctx context.Context, userID string) (insight.Facts, error)
}

// assembleFacts augments the base portfolio facts (from BuildFacts) with the contribution, the
// grounded FII universe, and the deterministically computed split — then returns the augmented
// facts plus the computed split so the engine can join the authoritative numbers onto the LLM's
// per-area explanations (the LLM explains the split, it never produces the numbers — FR-1053a).
// The base map is built fresh per request, so it is mutated in place.
func assembleFacts(base insight.Facts, contribution Contribution, universe []marketdata.FIIQuote) (insight.Facts, []AreaShare) {
	risk, _ := base["risk_profile"].(string) // "" when the profile is unset → moderate direction
	split := Split(allocationFromFacts(base), profile.RiskProfile(risk), contribution)

	base["contribution_centavos"] = contribution.Centavos()
	base["fii_universe"] = universeFacts(universe)
	base["suggested_split"] = splitFacts(split)
	return base, split
}

// allocationFromFacts reconstructs the per-class current value from the base facts
// (current_value_centavos + allocation_bps), avoiding a second dashboard read (BR-1052).
func allocationFromFacts(base insight.Facts) Allocation {
	total, _ := base["current_value_centavos"].(int64)
	byClass := map[dashboard.AssetClass]int64{}
	if bps, ok := base["allocation_bps"].(map[string]int); ok {
		for cls, share := range bps {
			byClass[dashboard.AssetClass(cls)] = money.ApplyBps(total, share)
		}
	}
	return Allocation{TotalCentavos: total, ByClass: byClass}
}

// universeFacts compacts the FII universe into grounding facts (ticker/sector/yield/price). Order
// is preserved from ListFIIUniverse (by ticker), so the fact set is deterministic.
func universeFacts(universe []marketdata.FIIQuote) []map[string]any {
	out := make([]map[string]any, 0, len(universe))
	for _, q := range universe {
		out = append(out, map[string]any{
			"ticker":             q.Ticker.String(),
			"sector":             string(q.Sector),
			"dividend_yield_bps": q.DividendYieldBps,
			"price_centavos":     q.PriceCentavos,
		})
	}
	return out
}

// splitFacts renders the computed split as integer facts (bps + centavos), never a float.
func splitFacts(split []AreaShare) []map[string]any {
	out := make([]map[string]any, 0, len(split))
	for _, a := range split {
		out = append(out, map[string]any{
			"area":                      string(a.Class),
			"suggested_share_bps":       a.SuggestedShareBps,
			"suggested_amount_centavos": a.SuggestedAmountCentavos,
		})
	}
	return out
}

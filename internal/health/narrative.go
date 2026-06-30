package health

import (
	"context"
	"strings"

	"github.com/biel-ferreira/yield-forge/internal/insight"
)

// TaskHealthScore is the insight.Task the gated Insighter reasons under when narrating the score
// (SPEC-005 is task-agnostic; this owns the health-score narrative).
const TaskHealthScore insight.Task = "health_score"

// addNarrative requests the gated "professor" narrative explaining the already-computed score, and
// attaches it (best-effort). It is emitted ONLY through the gated Insighter (FR-013/014) and is
// grounded in the computed score + breakdown + market — it NEVER changes the number (BR-1062). An
// LLM error or an empty/unexplained result degrades to NarrativeAvailable:false, score untouched.
func (s *Service) addNarrative(ctx context.Context, userID string, in Inputs, hs HealthScore) HealthScore {
	res, err := s.insighter.Generate(ctx, insight.InsightRequest{
		Facts:  narrativeFacts(in, hs),
		Task:   TaskHealthScore,
		UserID: userID,
	})
	if err != nil {
		return hs // degrade: the computed score + breakdown stand (FR-1063)
	}
	text := narrativeText(res)
	if text == "" {
		return hs
	}
	hs.Narrative = text
	hs.NarrativeAvailable = true
	hs.Disclaimer = res.Disclaimer
	return hs
}

// narrativeText extracts the gated prose from the result (the gate guarantees it is explained and
// order-free); empty when the result carries no usable text.
func narrativeText(res insight.InsightResult) string {
	if len(res.Insights) == 0 {
		return ""
	}
	first := res.Insights[0]
	if d := strings.TrimSpace(first.Detail); d != "" {
		return d
	}
	return strings.TrimSpace(first.Explanation)
}

// narrativeFacts grounds the narrative in the COMPUTED score + factor sub-scores + the market and
// allocation — so the LLM explains the number it is given, never invents one. Integers only.
func narrativeFacts(in Inputs, hs HealthScore) insight.Facts {
	factorScores := make(map[string]int, len(hs.Factors))
	for _, f := range hs.Factors {
		factorScores[string(f.Factor)] = f.Score
	}
	facts := insight.Facts{
		"health_score":                hs.Score,
		"factor_scores":               factorScores,
		"fii_value_centavos":          in.FIIValueCentavos,
		"fixed_income_value_centavos": in.FixedIncomeValueCentavos,
	}
	if in.HasProfile {
		facts["risk_profile"] = string(in.Risk)
	}
	if in.HasMacro {
		facts["selic_bps"] = in.SelicBps
	}
	return facts
}

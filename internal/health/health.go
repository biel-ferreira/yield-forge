package health

import (
	"errors"
	"fmt"
	"strings"
)

// ErrUnknownFactor marks an unrecognised factor name (SPEC-106). Check with errors.Is.
var ErrUnknownFactor = errors.New("unknown health factor")

// Factor is a health-score factor (SPEC-106 FR-1064). Closed enum.
type Factor string

const (
	FactorDiversification Factor = "diversification"
	FactorConcentration   Factor = "concentration"
	FactorLiquidity       Factor = "liquidity"
	FactorGoalAlignment   Factor = "goal_alignment"
	FactorRiskExposure    Factor = "risk_exposure"
)

// defaultWeightBps is each factor's default weight in basis points (SPEC-106 D3); the present
// factors are renormalised to sum 10000 when the profile is unset (goal/risk omitted).
var defaultWeightBps = map[Factor]int{
	FactorDiversification: 2500,
	FactorConcentration:   2500,
	FactorLiquidity:       1500,
	FactorGoalAlignment:   2000,
	FactorRiskExposure:    1500,
}

// ParseFactor normalises s (trim + lower-case) into a Factor, or returns ErrUnknownFactor via %w.
func ParseFactor(s string) (Factor, error) {
	switch f := Factor(strings.ToLower(strings.TrimSpace(s))); f {
	case FactorDiversification, FactorConcentration, FactorLiquidity, FactorGoalAlignment, FactorRiskExposure:
		return f, nil
	default:
		return "", fmt.Errorf("parse factor %q: %w", s, ErrUnknownFactor)
	}
}

// FactorScore is one factor's contribution: a 0–100 sub-score, its weight (bps), and a computed,
// reproducible explanation (SPEC-106 FR-1062).
type FactorScore struct {
	Factor      Factor
	Score       int // 0–100
	WeightBps   int
	Explanation string
}

// HealthScore is the computed result (SPEC-106 §6). Score is the integer weighted mean of the
// factor sub-scores. Narrative is the optional gated "professor" prose (Phase 3); it is additive
// and never affects Score — NarrativeAvailable is false when the LLM was unavailable.
type HealthScore struct {
	Score              int // 0–100
	Factors            []FactorScore
	Narrative          string
	NarrativeAvailable bool
	Disclaimer         string
}

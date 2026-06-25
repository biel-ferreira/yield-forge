package profile

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrInvalidObjective marks a string that is not a supported objective.
	ErrInvalidObjective = errors.New("invalid objective")
	// ErrNoObjectives marks a profile with an empty objective set (BR-1014).
	ErrNoObjectives = errors.New("at least one objective is required")
)

// Objective is a financial goal. Closed enum; a profile carries one or more (BR-1014).
type Objective string

const (
	ObjectiveRetirement         Objective = "retirement"
	ObjectivePassiveIncome      Objective = "passive_income"
	ObjectiveWealthPreservation Objective = "wealth_preservation"
	ObjectiveLongTermGrowth     Objective = "long_term_growth"
)

var validObjectives = map[Objective]bool{
	ObjectiveRetirement: true, ObjectivePassiveIncome: true,
	ObjectiveWealthPreservation: true, ObjectiveLongTermGrowth: true,
}

// ParseObjective normalizes (trim + lower) and validates s.
func ParseObjective(s string) (Objective, error) {
	o := Objective(strings.ToLower(strings.TrimSpace(s)))
	if !validObjectives[o] {
		return "", fmt.Errorf("parse objective %q: %w", s, ErrInvalidObjective)
	}
	return o, nil
}

// ParseObjectives parses raw strings into a deduplicated, non-empty objective set, in the
// order first seen (BR-1014). An unknown value returns ErrInvalidObjective; an empty result
// returns ErrNoObjectives.
func ParseObjectives(raw []string) ([]Objective, error) {
	seen := make(map[Objective]bool, len(raw))
	out := make([]Objective, 0, len(raw))
	for _, s := range raw {
		o, err := ParseObjective(s)
		if err != nil {
			return nil, err
		}
		if !seen[o] {
			seen[o] = true
			out = append(out, o)
		}
	}
	if len(out) == 0 {
		return nil, ErrNoObjectives
	}
	return out, nil
}

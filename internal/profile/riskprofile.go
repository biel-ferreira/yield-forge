package profile

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidRiskProfile marks a string that is not a supported risk profile (SPEC-101 FR-1011).
var ErrInvalidRiskProfile = errors.New("invalid risk profile")

// RiskProfile is the investor's risk tolerance. Closed enum (parse-don't-validate).
type RiskProfile string

const (
	RiskConservative RiskProfile = "conservative"
	RiskModerate     RiskProfile = "moderate"
	RiskAggressive   RiskProfile = "aggressive"
)

var validRiskProfiles = map[RiskProfile]bool{
	RiskConservative: true, RiskModerate: true, RiskAggressive: true,
}

// ParseRiskProfile normalizes (trim + lower) and validates s.
func ParseRiskProfile(s string) (RiskProfile, error) {
	rp := RiskProfile(strings.ToLower(strings.TrimSpace(s)))
	if !validRiskProfiles[rp] {
		return "", fmt.Errorf("parse risk profile %q: %w", s, ErrInvalidRiskProfile)
	}
	return rp, nil
}

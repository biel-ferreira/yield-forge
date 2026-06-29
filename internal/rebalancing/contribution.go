package rebalancing

import (
	"errors"
	"fmt"
)

// ErrInvalidContribution is returned when a contribution amount is not strictly positive
// (SPEC-105 BR-1051). Check with errors.Is.
var ErrInvalidContribution = errors.New("contribution must be positive")

// Contribution is the new money to allocate, in int64 centavos — parse-don't-validate: an invalid
// amount is unrepresentable (SPEC-105 FR-1051/BR-1051). It never crosses the JSON boundary as a float.
type Contribution struct {
	centavos int64
}

// ParseContribution builds a Contribution, rejecting any non-positive amount.
func ParseContribution(centavos int64) (Contribution, error) {
	if centavos <= 0 {
		return Contribution{}, fmt.Errorf("parse contribution: %w", ErrInvalidContribution)
	}
	return Contribution{centavos: centavos}, nil
}

// Centavos returns the amount in centavos.
func (c Contribution) Centavos() int64 { return c.centavos }

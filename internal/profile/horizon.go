package profile

import (
	"errors"
	"fmt"
)

// ErrInvalidHorizon marks an investment horizon outside the supported range (SPEC-101 BR-1015).
var ErrInvalidHorizon = errors.New("invalid investment horizon")

// Horizon bounds (whole years). The PRD's 5/10/20 are examples, not a fixed enum.
const (
	MinHorizonYears = 1
	MaxHorizonYears = 50
)

// Horizon is the investment horizon in whole years (parse-don't-validate).
type Horizon struct{ years int }

// ParseHorizon validates years against [MinHorizonYears, MaxHorizonYears].
func ParseHorizon(years int) (Horizon, error) {
	if years < MinHorizonYears || years > MaxHorizonYears {
		return Horizon{}, fmt.Errorf("parse horizon %d: %w (want %d-%d years)", years, ErrInvalidHorizon, MinHorizonYears, MaxHorizonYears)
	}
	return Horizon{years: years}, nil
}

// Years returns the horizon in years. The zero Horizon is 0 (never produced by ParseHorizon).
func (h Horizon) Years() int { return h.years }

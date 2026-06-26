package portfolio

import (
	"errors"
	"fmt"
)

// ErrInvalidQuantity marks a non-positive holding quantity (SPEC-102 BR-1023).
var ErrInvalidQuantity = errors.New("invalid quantity")

// Quantity is a whole number of FII cotas, strictly positive (parse-don't-validate). FII
// cotas are integral units (D5), so a fractional or non-positive quantity is unrepresentable.
type Quantity struct{ value int }

// ParseQuantity validates that n is a positive whole number.
func ParseQuantity(n int) (Quantity, error) {
	if n <= 0 {
		return Quantity{}, fmt.Errorf("parse quantity %d: %w", n, ErrInvalidQuantity)
	}
	return Quantity{value: n}, nil
}

// Value returns the quantity. The zero Quantity is 0 (never produced by ParseQuantity).
func (q Quantity) Value() int { return q.value }

// Package money centralizes the conversion of decimal strings into integer minor units
// with ONE documented rounding rule — half-up — so monetary and rate values are
// deterministic and reproducible (CLAUDE.md: "same inputs → same Health Score"). Money is
// never float64: prices are int64 centavos and rates are integer basis points, all built
// here from the strings the market-data providers return (SPEC-006 BR-604).
package money

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ErrInvalidDecimal marks a string that is not a parseable decimal number.
var ErrInvalidDecimal = errors.New("invalid decimal number")

// DecimalToMinor parses a decimal number into an int64 scaled by 10^scale, rounding
// half-up. It accepts both Brazilian and plain forms: if a comma is present, dots are
// treated as thousands separators and the comma as the decimal point ("1.234,56" →
// 123456 at scale 2); otherwise a dot is the decimal point ("0.11" → 11 at scale 2).
// Callers pass a clean numeric string (e.g. strip a trailing '%'). Rounding is half-up by
// magnitude, so a negative rounds away from zero ("-0,005" at scale 2 -> -1). Market-data
// values are non-negative; this only matters if money is reused for signed amounts.
//
//	DecimalToMinor("15,75", 2) -> 1575   // R$15,75 -> centavos
//	DecimalToMinor("8,50", 2)  -> 850    // 8.50%   -> basis points
//	DecimalToMinor("0,95", 4)  -> 9500   // P/VP ratio -> ratio basis points
func DecimalToMinor(s string, scale int) (int64, error) {
	if scale < 0 {
		return 0, fmt.Errorf("money: negative scale %d", scale)
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("money: empty: %w", ErrInvalidDecimal)
	}

	neg := false
	switch s[0] {
	case '-':
		neg, s = true, s[1:]
	case '+':
		s = s[1:]
	}
	// Normalize to a '.'-decimal with no thousands separators.
	if strings.Contains(s, ",") {
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	}

	intPart, fracPart, _ := strings.Cut(s, ".")
	if intPart == "" && fracPart == "" {
		// No digits at all — e.g. "", a lone sign, or ".".
		return 0, fmt.Errorf("money: %q: %w", s, ErrInvalidDecimal)
	}
	if intPart == "" {
		intPart = "0"
	}
	if !isDigits(intPart) || !isDigits(fracPart) {
		return 0, fmt.Errorf("money: %q: %w", s, ErrInvalidDecimal)
	}

	// Round half-up to the requested scale.
	roundUp := false
	if len(fracPart) > scale {
		roundUp = fracPart[scale] >= '5'
		fracPart = fracPart[:scale]
	} else {
		fracPart += strings.Repeat("0", scale-len(fracPart))
	}

	val, err := strconv.ParseInt(intPart+fracPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("money: %q: %w", s, ErrInvalidDecimal)
	}
	if roundUp {
		if val == math.MaxInt64 { // the carry would overflow int64
			return 0, fmt.Errorf("money: %q: overflow: %w", s, ErrInvalidDecimal)
		}
		val++
	}
	if neg {
		val = -val
	}
	return val, nil
}

// isDigits reports whether s is all ASCII digits. The empty string counts as digits-only
// (a number may have no fractional part).
func isDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

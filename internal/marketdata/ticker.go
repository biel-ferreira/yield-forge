package marketdata

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ErrInvalidTicker marks a string that is not a well-formed B3 FII ticker (SPEC-006 §6).
var ErrInvalidTicker = errors.New("invalid ticker")

// tickerPattern matches a B3 ticker: four letters then one or two digits (FIIs are the
// "11" form, e.g. HGLG11, KNRI11, MXRF11).
var tickerPattern = regexp.MustCompile(`^[A-Z]{4}[0-9]{1,2}$`)

// Ticker is a validated B3 ticker (parse-don't-validate: an invalid Ticker cannot exist).
type Ticker struct{ value string }

// ParseTicker normalizes (trim + upper) and validates s, returning ErrInvalidTicker when
// it is not a well-formed B3 ticker.
func ParseTicker(s string) (Ticker, error) {
	v := strings.ToUpper(strings.TrimSpace(s))
	if !tickerPattern.MatchString(v) {
		return Ticker{}, fmt.Errorf("parse ticker %q: %w", s, ErrInvalidTicker)
	}
	return Ticker{value: v}, nil
}

// MustParseTicker is ParseTicker for tests/fixtures; it panics on an invalid input.
func MustParseTicker(s string) Ticker {
	t, err := ParseTicker(s)
	if err != nil {
		panic(err)
	}
	return t
}

// String returns the canonical ticker (e.g. "HGLG11"). The zero Ticker is "".
func (t Ticker) String() string { return t.value }

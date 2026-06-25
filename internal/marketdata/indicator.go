package marketdata

import (
	"errors"
	"fmt"
)

// ErrInvalidIndicator marks a string that is not one of the supported macro indicators.
var ErrInvalidIndicator = errors.New("invalid macro indicator")

// Indicator is a supported Brazilian macro indicator (SPEC-006 FR-007). Closed enum.
type Indicator string

const (
	IndicatorSELIC Indicator = "selic" // policy interest rate
	IndicatorIPCA  Indicator = "ipca"  // official inflation index
	IndicatorCDI   Indicator = "cdi"   // interbank deposit rate
	IndicatorIFIX  Indicator = "ifix"  // B3 real-estate-fund index
)

// AllIndicators is the fixed set the ingestion worker refreshes each run.
var AllIndicators = []Indicator{IndicatorSELIC, IndicatorIPCA, IndicatorCDI, IndicatorIFIX}

var validIndicators = map[Indicator]bool{
	IndicatorSELIC: true, IndicatorIPCA: true, IndicatorCDI: true, IndicatorIFIX: true,
}

// ParseIndicator validates s (case-insensitive) against the supported set.
func ParseIndicator(s string) (Indicator, error) {
	ind := Indicator(toLowerASCII(s))
	if !validIndicators[ind] {
		return "", fmt.Errorf("parse indicator %q: %w", s, ErrInvalidIndicator)
	}
	return ind, nil
}

// Unit is the measurement unit a MacroIndicator value is expressed in. Rates (SELIC, IPCA,
// CDI) are basis points; IFIX is an index level in points (SPEC-006 §6, BR-604).
type Unit string

const (
	UnitBps    Unit = "bps"
	UnitPoints Unit = "points"
)

// DefaultUnit is the unit each indicator is stored in.
func (i Indicator) DefaultUnit() Unit {
	if i == IndicatorIFIX {
		return UnitPoints
	}
	return UnitBps
}

// toLowerASCII lowercases A–Z without touching anything else (indicator names are ASCII).
func toLowerASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}

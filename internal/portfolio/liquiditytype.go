package portfolio

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidLiquidityType marks a string that is not a supported liquidity type (SPEC-102 D3).
var ErrInvalidLiquidityType = errors.New("invalid liquidity type")

// LiquidityType is how liquid a fixed-income holding is (D3). Closed enum. The instrument
// name (CDB, Tesouro Direto, "caixinha", …) lives in the holding's Name field; this captures
// only whether the money is available daily or locked until maturity.
type LiquidityType string

const (
	LiquidityDaily      LiquidityType = "daily"       // redeemable any day (e.g. a daily-liquidity caixinha)
	LiquidityAtMaturity LiquidityType = "at_maturity" // locked until maturity_date
)

var validLiquidityTypes = map[LiquidityType]bool{
	LiquidityDaily: true, LiquidityAtMaturity: true,
}

// ParseLiquidityType normalizes (trim + lower) and validates s.
func ParseLiquidityType(s string) (LiquidityType, error) {
	lt := LiquidityType(strings.ToLower(strings.TrimSpace(s)))
	if !validLiquidityTypes[lt] {
		return "", fmt.Errorf("parse liquidity type %q: %w", s, ErrInvalidLiquidityType)
	}
	return lt, nil
}

// RequiresMaturity reports whether a holding of this liquidity type must carry a maturity
// date (at-maturity instruments do; daily-liquidity ones do not).
func (l LiquidityType) RequiresMaturity() bool { return l == LiquidityAtMaturity }

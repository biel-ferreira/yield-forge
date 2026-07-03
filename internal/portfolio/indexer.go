package portfolio

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidIndexer marks a string that is not a supported rate indexer (SPEC-109 FR-1091).
var ErrInvalidIndexer = errors.New("invalid rate indexer")

// Indexer is how a fixed-income holding's rate is quoted (SPEC-109). Closed enum. The
// AnnualRateBps field on FixedIncomeHolding is reinterpreted per indexer:
//   - Prefixado:      AnnualRateBps IS the flat annual rate (unchanged pre-SPEC-109 behavior).
//   - CDIPercentual:  AnnualRateBps is the percentage of CDI, in bps-of-percent (12000 = 120%).
//   - IPCASpread:     AnnualRateBps is the spread over IPCA, in bps (580 = +5.80%).
type Indexer string

const (
	IndexerPrefixado     Indexer = "prefixado"      // flat annual rate
	IndexerCDIPercentual Indexer = "cdi_percentual" // % of CDI (pós-fixado)
	IndexerIPCASpread    Indexer = "ipca_spread"    // IPCA + spread (híbrido)
)

var validIndexers = map[Indexer]bool{
	IndexerPrefixado: true, IndexerCDIPercentual: true, IndexerIPCASpread: true,
}

// ParseIndexer normalizes (trim + lower) and validates s. An empty string defaults to
// Prefixado (SPEC-109 BR-1093 backward compatibility — existing holdings/callers that never
// supply an indexer keep today's flat-rate behavior).
func ParseIndexer(s string) (Indexer, error) {
	trimmed := strings.ToLower(strings.TrimSpace(s))
	if trimmed == "" {
		return IndexerPrefixado, nil
	}
	idx := Indexer(trimmed)
	if !validIndexers[idx] {
		return "", fmt.Errorf("parse indexer %q: %w", s, ErrInvalidIndexer)
	}
	return idx, nil
}

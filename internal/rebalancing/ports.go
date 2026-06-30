package rebalancing

import (
	"context"

	"github.com/biel-ferreira/yield-forge/internal/marketdata"
)

// UniverseReader supplies the grounded FII candidate universe (SPEC-105 FR-1054) — the real
// tickers the assistant may name. Consumer-defined (accept interfaces); satisfied at the wiring
// edge by the market-data Postgres adapter's ListFIIUniverse. The grounding guard validates any
// LLM-named candidate against this set, so a hallucinated ticker never reaches the user (BR-1053).
type UniverseReader interface {
	ListFIIUniverse(ctx context.Context) ([]marketdata.FIIQuote, error)
}

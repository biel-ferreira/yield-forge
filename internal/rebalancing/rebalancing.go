package rebalancing

import "github.com/biel-ferreira/yield-forge/internal/insight"

// TaskRebalancing is the insight.Task the gated Insighter reasons under for this feature (SPEC-005
// is task-agnostic; SPEC-104 owns insight/market tasks, this owns rebalancing).
const TaskRebalancing insight.Task = "rebalancing"

// Categories the engine tags gated items with before splitting them in the response (SPEC-105 D3).
const (
	categoryArea      = "area"
	categoryCandidate = "candidate"
)

// Area is one suggested allocation area: the computed split (FR-1053a) joined to the gated
// explanation from the Insighter (FR-013). Class is the asset class ("fii" / "fixed_income").
type Area struct {
	Class                   string
	SuggestedShareBps       int
	SuggestedAmountCentavos int64
	Title                   string
	Detail                  string
	Explanation             string
}

// Candidate is a named asset worth a look, grounded in the market-data universe (FR-1054). The
// illustrative within-area share is 0 unless the caller opted in (D6). Naming a candidate is a
// consideration, never an order (FR-014).
type Candidate struct {
	Ticker               string
	Sector               string
	Title                string
	Detail               string
	Explanation          string
	IllustrativeShareBps int // 0 unless include_asset_shares was requested
}

// Rebalancing is the engine's aggregate result (SPEC-105 §6). Available is false when the LLM was
// fully unavailable (areas/candidates empty); the computed split is still meaningful but the
// guidance text is not produced.
type Rebalancing struct {
	Areas      []Area
	Candidates []Candidate
	Disclaimer string
	Available  bool
}

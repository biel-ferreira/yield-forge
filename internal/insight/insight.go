package insight

import "errors"

// Domain errors (sentinels; check with errors.Is).
//
// ErrAdviceDetected and ErrMissingExplanation are the binding-guard rejections: the
// gate fails closed rather than letting an unexplained or order-bearing output reach a
// user (SPEC-005 FR-013/FR-014, BR-506). ErrInsightsUnavailable is the graceful-
// degradation signal a caller renders as "insights temporarily unavailable".
var (
	ErrMissingExplanation  = errors.New("insight missing explanation")
	ErrAdviceDetected      = errors.New("output contains a transaction order")
	ErrInsightsUnavailable = errors.New("insights temporarily unavailable")
	ErrInsufficientFacts   = errors.New("insufficient facts to generate insights")
)

// Facts is the deterministic, structured snapshot the LLM reasons over (BR-502 — facts
// are computed, not generated). Its concrete shape is owned by the Fact Builder
// (SPEC-104); this port treats it opaquely: it is serialized into the prompt and hashed
// into the cache key. A JSON object keeps it forward-compatible without coupling the
// port to the insight domain.
type Facts map[string]any

// Task names what to reason about. The specific tasks (concentration, allocation,
// market-context, …) are defined by SPEC-104; SPEC-005 is task-agnostic.
type Task string

// Insight is one explainable observation, framed as an area/consideration — never an
// order (FR-014). Explanation is required (FR-013).
type Insight struct {
	Category    string
	Title       string
	Detail      string
	Explanation string
}

// InsightRequest is a facts-grounded generation request. UserID (from the request
// context, SPEC-003 BR-304) scopes the cache key so one user's cache never serves
// another.
type InsightRequest struct {
	Facts  Facts
	Task   Task
	UserID string
}

// InsightResult is a set of gated insights plus the mandatory non-advice disclaimer
// (FR-014). Every Insight in a result that left the gate carries an Explanation.
type InsightResult struct {
	Insights   []Insight
	Disclaimer string
}

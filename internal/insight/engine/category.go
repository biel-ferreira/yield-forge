package engine

import "github.com/biel-ferreira/yield-forge/internal/insight"

// Category is an insight category (SPEC-104 FR-1042/1043/1044). Each maps 1:1 to an
// insight.Task: the engine builds one fact set and calls the Insighter once per category, so
// the SPEC-005 cache keys by (user, task, facts) and repeats are cheap (D2).
type Category string

const (
	CategoryPortfolio     Category = "portfolio"      // concentration, sector imbalance, risk, diversification (FR-008)
	CategoryAllocation    Category = "allocation"     // allocation vs profile, single-sector/asset concentration (FR-009)
	CategoryMarketContext Category = "market_context" // macro conditions tied to the portfolio (FR-010)
)

// AllCategories is the fixed order the engine generates and the response presents.
var AllCategories = []Category{CategoryPortfolio, CategoryAllocation, CategoryMarketContext}

// Task is the insight.Task the Insighter reasons under for this category (same string).
func (c Category) Task() insight.Task { return insight.Task(c) }

// Insights is the engine's aggregate result across categories (SPEC-104 §6). Available is
// false when the LLM was fully unavailable (Items empty); a partial result keeps Available
// true with the categories that succeeded (FR-1047, D5).
type Insights struct {
	Items      []insight.Insight
	Disclaimer string
	Available  bool
}

package chat

import (
	"context"

	"github.com/biel-ferreira/yield-forge/internal/insight"
)

// The chat engine grounds each turn through small consumer interfaces (accept interfaces), satisfied
// at the wiring edge by the SPEC-104 Fact Builder, the SPEC-105 rebalancer, and the SPEC-107
// projections. Each returns a deterministic insight.Facts snapshot — facts are computed, never
// generated (BR-1081); none of these invoke the LLM, so a chat turn never double-generates (D5).

// FactSource builds the general grounding facts for a turn (the SPEC-104 Fact Builder's BuildFacts).
type FactSource interface {
	BuildFacts(ctx context.Context, userID string) (insight.Facts, error)
}

// ContributionFactSource grounds a "tenho R$X pra aportar" turn (the SPEC-105 rebalancer's
// deterministic split facts — no per-area LLM). A runtime error degrades the turn to the general facts.
type ContributionFactSource interface {
	BuildContributionFacts(ctx context.Context, userID string, amountCentavos int64) (insight.Facts, error)
}

// ProjectionFactSource grounds a "daqui a N anos" / passive-income turn (the SPEC-107 projections —
// already LLM-free). A runtime error degrades the turn to the general facts.
type ProjectionFactSource interface {
	BuildProjectionFacts(ctx context.Context, userID string, monthlyContributionCentavos int64, horizonYears int) (insight.Facts, error)
}

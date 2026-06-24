package insight

import "context"

// Insighter is the single seam through which the application invokes an LLM
// (SPEC-005 FR-501 / FR-018). Implemented by provider adapters (ollama/, groq/), the
// deterministic fake, and the gate/cache/observability decorators — all behind this one
// interface, so the provider is swappable by config and the guards are inescapable.
//
// Generate reasons over req.Facts and returns gated, explainable, advice-free insights,
// or an error: ErrInsufficientFacts (nothing to reason over), ErrInsightsUnavailable
// (provider down/rate-limited — degrade), or a guard rejection from the gate decorator.
type Insighter interface {
	Generate(ctx context.Context, req InsightRequest) (InsightResult, error)
}

// Cache stores gated InsightResults keyed by a hash of the request (facts + task +
// user). The default implementation is in-memory (SPEC-005 D4); a persistent backing is
// a later drop-in behind this port. A cache miss (or any internal cache error) returns
// found=false so the caller falls through to the LLM — caching never blocks generation.
type Cache interface {
	Get(ctx context.Context, key string) (result InsightResult, found bool)
	Set(ctx context.Context, key string, result InsightResult)
}

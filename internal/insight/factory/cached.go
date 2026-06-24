package factory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/biel-ferreira/yield-forge/internal/insight"
)

// cached wraps an Insighter with a result cache (SPEC-005 FR-506). It wraps the GATED
// insighter, so a cached result is already explained + advice-free. Errors
// (degradations, gate rejections) are never cached. cache_hit is recorded on the active
// span (set up by the observed decorator).
type cached struct {
	next  insight.Insighter
	cache insight.Cache
}

func (c cached) Generate(ctx context.Context, req insight.InsightRequest) (insight.InsightResult, error) {
	key := cacheKey(req)

	if res, ok := c.cache.Get(ctx, key); ok {
		trace.SpanFromContext(ctx).SetAttributes(attribute.Bool("insight.cache_hit", true))
		return res, nil
	}
	trace.SpanFromContext(ctx).SetAttributes(attribute.Bool("insight.cache_hit", false))

	res, err := c.next.Generate(ctx, req)
	if err != nil {
		return insight.InsightResult{}, err // never cache errors/degradations/rejections
	}
	c.cache.Set(ctx, key, res)
	return res, nil
}

// cacheKey hashes the user, task, and facts into a stable key. The user is included so
// one user's cache never serves another (SPEC-003 BR-304); json.Marshal sorts map keys,
// so equal facts hash equally regardless of construction order.
func cacheKey(req insight.InsightRequest) string {
	facts, _ := json.Marshal(req.Facts)
	h := sha256.New()
	h.Write([]byte(req.UserID))
	h.Write([]byte{0})
	h.Write([]byte(req.Task))
	h.Write([]byte{0})
	h.Write(facts)
	return hex.EncodeToString(h.Sum(nil))
}

package factory

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/biel-ferreira/yield-forge/internal/insight"
	"github.com/biel-ferreira/yield-forge/internal/platform/config"
)

// countingInsighter counts calls so cache hits/misses are observable.
type countingInsighter struct {
	calls  int
	result insight.InsightResult
	err    error
}

func (c *countingInsighter) Generate(context.Context, insight.InsightRequest) (insight.InsightResult, error) {
	c.calls++
	return c.result, c.err
}

// fakeClock is a mutable Clock for deterministic TTL tests.
type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time { return c.t }

func okResult() insight.InsightResult {
	return insight.InsightResult{Insights: []insight.Insight{{Title: "t", Explanation: "e"}}}
}

func req(user string) insight.InsightRequest {
	return insight.InsightRequest{Facts: insight.Facts{"x": 1}, Task: "overview", UserID: user}
}

func newCached(stub insight.Insighter, ttl time.Duration, clk *fakeClock) cached {
	return cached{next: stub, cache: newMemCache(16, ttl, clk)}
}

func TestCached_MissThenHit(t *testing.T) {
	stub := &countingInsighter{result: okResult()}
	c := newCached(stub, time.Hour, &fakeClock{t: time.Unix(1_700_000_000, 0).UTC()})

	_, err := c.Generate(context.Background(), req("u1"))
	require.NoError(t, err)
	_, err = c.Generate(context.Background(), req("u1"))
	require.NoError(t, err)
	require.Equal(t, 1, stub.calls, "an identical second request is served from cache")
}

func TestCached_DifferentUserMisses(t *testing.T) {
	stub := &countingInsighter{result: okResult()}
	c := newCached(stub, time.Hour, &fakeClock{t: time.Unix(1_700_000_000, 0).UTC()})

	_, _ = c.Generate(context.Background(), req("u1"))
	_, _ = c.Generate(context.Background(), req("u2"))
	require.Equal(t, 2, stub.calls, "different users do not share a cache entry (BR-304)")
}

func TestCached_TTLExpiry(t *testing.T) {
	clk := &fakeClock{t: time.Unix(1_700_000_000, 0).UTC()}
	stub := &countingInsighter{result: okResult()}
	c := newCached(stub, time.Minute, clk)

	_, _ = c.Generate(context.Background(), req("u1"))
	clk.t = clk.t.Add(2 * time.Minute) // past the TTL
	_, _ = c.Generate(context.Background(), req("u1"))
	require.Equal(t, 2, stub.calls, "an expired entry is a miss")
}

func TestCached_DoesNotCacheErrors(t *testing.T) {
	stub := &countingInsighter{err: insight.ErrInsightsUnavailable}
	c := newCached(stub, time.Hour, &fakeClock{t: time.Unix(1_700_000_000, 0).UTC()})

	_, _ = c.Generate(context.Background(), req("u1"))
	_, _ = c.Generate(context.Background(), req("u1"))
	require.Equal(t, 2, stub.calls, "errors/degradations are never cached")
}

func TestNew_FakeProvider_EndToEnd(t *testing.T) {
	cfg := config.Config{InsighterProvider: "fake", InsighterCacheSize: 16, InsighterCacheTTL: time.Hour}
	in := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), &fakeClock{t: time.Unix(1_700_000_000, 0).UTC()})

	res, err := in.Generate(context.Background(), req("u1"))
	require.NoError(t, err)
	require.Equal(t, insight.Disclaimer, res.Disclaimer, "the gate is in the composed chain")
	require.Len(t, res.Insights, 1)
}

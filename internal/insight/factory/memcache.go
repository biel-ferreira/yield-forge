// Package factory is the insight composition root (SPEC-005): it wires the provider
// adapters, the guard gates, the cache, and the AI observability into a single
// Insighter from config. It is the only part of the insight area that imports the
// adapters and OpenTelemetry — the insight core and the adapters stay pure.
package factory

import (
	"container/list"
	"context"
	"sync"
	"time"

	"github.com/biel-ferreira/yield-forge/internal/insight"
	"github.com/biel-ferreira/yield-forge/internal/platform/clock"
)

// memCache is an in-memory LRU cache with a per-entry TTL (SPEC-005 D4), safe for
// concurrent use. It implements insight.Cache. TTL is evaluated against the injected
// Clock so expiry is deterministic in tests.
type memCache struct {
	mu    sync.Mutex
	ttl   time.Duration
	size  int
	clock clock.Clock
	ll    *list.List // front = most recently used
	items map[string]*list.Element
}

type cacheEntry struct {
	key     string
	result  insight.InsightResult
	expires time.Time
}

func newMemCache(size int, ttl time.Duration, clk clock.Clock) *memCache {
	return &memCache{
		ttl:   ttl,
		size:  size,
		clock: clk,
		ll:    list.New(),
		items: make(map[string]*list.Element),
	}
}

func (c *memCache) Get(_ context.Context, key string) (insight.InsightResult, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.items[key]
	if !ok {
		return insight.InsightResult{}, false
	}
	entry := el.Value.(*cacheEntry)
	if !c.clock.Now().Before(entry.expires) { // expired
		c.removeElement(el)
		return insight.InsightResult{}, false
	}
	c.ll.MoveToFront(el)
	return entry.result, true
}

func (c *memCache) Set(_ context.Context, key string, result insight.InsightResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	expires := c.clock.Now().Add(c.ttl)
	if el, ok := c.items[key]; ok {
		entry := el.Value.(*cacheEntry)
		entry.result = result
		entry.expires = expires
		c.ll.MoveToFront(el)
		return
	}
	c.items[key] = c.ll.PushFront(&cacheEntry{key: key, result: result, expires: expires})
	if c.ll.Len() > c.size {
		if oldest := c.ll.Back(); oldest != nil {
			c.removeElement(oldest)
		}
	}
}

func (c *memCache) removeElement(el *list.Element) {
	c.ll.Remove(el)
	delete(c.items, el.Value.(*cacheEntry).key)
}

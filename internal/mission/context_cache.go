package mission

import (
	"fmt"
	"sync"
	"time"
)

// CacheState describes a cache lookup without changing materialization truth.
type CacheState string

const (
	// CacheHit means a valid entry matched the requested identity.
	CacheHit CacheState = "hit"
	// CacheMiss means no entry matched the requested identity.
	CacheMiss CacheState = "miss"
	// CacheStale means an entry exceeded its retention period.
	CacheStale CacheState = "stale"
	// CacheCorrupt means an entry could not be used safely.
	CacheCorrupt CacheState = "corrupt"
	// CacheUnavailable means no cache was configured for materialization.
	CacheUnavailable CacheState = "unavailable"
)

// ContextCachePolicy limits cache scope and retention to a local workspace.
type ContextCachePolicy struct {
	MaxEntries int
	Retention  time.Duration
}

// ContextCache is an in-process, workspace-local cache. It does not share
// entries across processes, runtimes, or providers.
type ContextCache struct {
	mu      sync.Mutex
	policy  ContextCachePolicy
	now     func() time.Time
	entries map[string]contextCacheEntry
}

type contextCacheEntry struct {
	identity ContextIdentity
	content  string
	storedAt time.Time
}

// NewContextCache creates a bounded cache with an explicit retention period.
func NewContextCache(policy ContextCachePolicy, now func() time.Time) (*ContextCache, error) {
	if policy.MaxEntries <= 0 || policy.Retention <= 0 {
		return nil, fmt.Errorf("context cache: max entries and retention must be positive")
	}
	if now == nil {
		now = time.Now
	}
	return &ContextCache{policy: policy, now: now, entries: make(map[string]contextCacheEntry)}, nil
}

// Materialize returns a matching cached value or invokes source materialization.
// A cache problem never returns content for a different identity.
func Materialize(cache *ContextCache, identity ContextIdentity, source func() (string, error)) (string, CacheState, error) {
	if source == nil {
		return "", CacheUnavailable, fmt.Errorf("context cache: source materializer is required")
	}
	state := CacheUnavailable
	if cache != nil {
		content, cacheState := cache.get(identity)
		if cacheState == CacheHit {
			return content, cacheState, nil
		}
		state = cacheState
	}
	content, err := source()
	if err != nil {
		return "", CacheUnavailable, fmt.Errorf("context cache: materialize source: %w", err)
	}
	if cache == nil {
		return content, CacheUnavailable, nil
	}
	cache.put(identity, content)
	return content, state, nil
}

func (c *ContextCache) get(identity ContextIdentity) (string, CacheState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[identity.Key]
	if !ok {
		return "", CacheMiss
	}
	if entry.identity != identity || entry.content == "" {
		delete(c.entries, identity.Key)
		return "", CacheCorrupt
	}
	if c.now().Sub(entry.storedAt) > c.policy.Retention {
		delete(c.entries, identity.Key)
		return "", CacheStale
	}
	return entry.content, CacheHit
}

func (c *ContextCache) put(identity ContextIdentity, content string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= c.policy.MaxEntries {
		for key := range c.entries {
			delete(c.entries, key)
			break
		}
	}
	c.entries[identity.Key] = contextCacheEntry{identity: identity, content: content, storedAt: c.now()}
}

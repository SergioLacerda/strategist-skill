package mission

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMaterializeUsesMatchingCacheAndFallsBackSafely(t *testing.T) {
	now := time.Unix(100, 0)
	cache, err := NewContextCache(ContextCachePolicy{MaxEntries: 1, Retention: 24 * time.Hour}, func() time.Time { return now })
	require.NoError(t, err)
	identity := testContextIdentity(t, "a")
	calls := 0
	source := func() (string, error) { calls++; return "fresh", nil }

	content, state, err := Materialize(cache, identity, source)
	require.NoError(t, err)
	require.Equal(t, "fresh", content)
	require.Equal(t, CacheMiss, state)
	content, state, err = Materialize(cache, identity, source)
	require.NoError(t, err)
	require.Equal(t, "fresh", content)
	require.Equal(t, CacheHit, state)
	require.Equal(t, 1, calls)

	content, state, err = Materialize(cache, testContextIdentity(t, "b"), source)
	require.NoError(t, err)
	require.Equal(t, "fresh", content)
	require.Equal(t, CacheMiss, state)
	require.Equal(t, 2, calls)
}

func TestMaterializeHandlesStaleUnavailableAndSourceFailure(t *testing.T) {
	now := time.Unix(100, 0)
	cache, err := NewContextCache(ContextCachePolicy{MaxEntries: 1, Retention: time.Hour}, func() time.Time { return now })
	require.NoError(t, err)
	identity := testContextIdentity(t, "a")
	_, _, err = Materialize(cache, identity, func() (string, error) { return "old", nil })
	require.NoError(t, err)
	now = now.Add(2 * time.Hour)
	content, state, err := Materialize(cache, identity, func() (string, error) { return "new", nil })
	require.NoError(t, err)
	require.Equal(t, "new", content)
	require.Equal(t, CacheStale, state)

	_, state, err = Materialize(nil, identity, func() (string, error) { return "uncached", nil })
	require.NoError(t, err)
	require.Equal(t, CacheUnavailable, state)
	_, _, err = Materialize(cache, testContextIdentity(t, "failure"), func() (string, error) { return "", errors.New("source unavailable") })
	require.ErrorContains(t, err, "source unavailable")
}

func testContextIdentity(t *testing.T, suffix string) ContextIdentity {
	t.Helper()
	identity, err := NewContextIdentity(ContextIdentityInput{
		SourceDigest: "source-" + suffix, RuntimeDigest: "runtime", ProviderDigest: "provider",
		PolicyDigest: "policy", ContentDigest: "content",
	})
	require.NoError(t, err)
	return identity
}

func TestMaterializeIsSafeUnderConcurrentAccess(t *testing.T) {
	cache, err := NewContextCache(ContextCachePolicy{MaxEntries: 2, Retention: time.Hour}, time.Now)
	require.NoError(t, err)

	const workers, rounds = 16, 50
	suffixes := []string{"a", "b", "c", "d"}
	identities := make(map[string]ContextIdentity, len(suffixes))
	for _, suffix := range suffixes {
		identities[suffix] = testContextIdentity(t, suffix) // built here: require must not run in goroutines
	}
	errs := runConcurrentMaterialization(cache, identities, suffixes, workers, rounds)
	for err := range errs {
		require.NoError(t, err)
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	require.LessOrEqual(t, len(cache.entries), 2, "cache must stay bounded under concurrency")
}

func runConcurrentMaterialization(cache *ContextCache, identities map[string]ContextIdentity, suffixes []string, workers, rounds int) chan error {
	var wg sync.WaitGroup
	errs := make(chan error, workers*rounds)
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go materializeWorker(cache, identities, suffixes, rounds, worker, &wg, errs)
	}
	wg.Wait()
	close(errs)
	return errs
}

func materializeWorker(cache *ContextCache, identities map[string]ContextIdentity, suffixes []string, rounds, worker int, wg *sync.WaitGroup, errs chan<- error) {
	defer wg.Done()
	for round := 0; round < rounds; round++ {
		suffix := suffixes[(worker+round)%len(suffixes)]
		identity := identities[suffix]
		content, _, err := Materialize(cache, identity, func() (string, error) { return "content-" + suffix, nil })
		if err != nil {
			errs <- err
			continue
		}
		if content != "content-"+suffix {
			errs <- fmt.Errorf("identity %s returned %q", suffix, content)
		}
	}
}

func TestMaterializeReportsCorruptEntryAndRecovers(t *testing.T) {
	cache, err := NewContextCache(ContextCachePolicy{MaxEntries: 2, Retention: time.Hour}, time.Now)
	require.NoError(t, err)
	identity := testContextIdentity(t, "a")
	other := testContextIdentity(t, "b")
	cache.entries[identity.Key] = contextCacheEntry{identity: other, content: "wrong", storedAt: time.Now()}

	content, state, err := Materialize(cache, identity, func() (string, error) { return "truth", nil })
	require.NoError(t, err)
	require.Equal(t, "truth", content)
	require.Equal(t, CacheCorrupt, state)

	content, state, err = Materialize(cache, identity, func() (string, error) { return "unused", nil })
	require.NoError(t, err)
	require.Equal(t, "truth", content)
	require.Equal(t, CacheHit, state)
}

// TestDisablingCacheRollsBackToUncachedMaterialization is the migration-rollback
// fixture: switching the cache off (nil) must keep serving source truth and must
// not delete or corrupt the entries the cache already holds.
func TestDisablingCacheRollsBackToUncachedMaterialization(t *testing.T) {
	cache, err := NewContextCache(ContextCachePolicy{MaxEntries: 2, Retention: time.Hour}, time.Now)
	require.NoError(t, err)
	identity := testContextIdentity(t, "a")
	_, _, err = Materialize(cache, identity, func() (string, error) { return "cached", nil })
	require.NoError(t, err)

	calls := 0
	source := func() (string, error) { calls++; return "source-truth", nil }
	for i := 0; i < 2; i++ {
		content, state, err := Materialize(nil, identity, source)
		require.NoError(t, err)
		require.Equal(t, "source-truth", content)
		require.Equal(t, CacheUnavailable, state)
	}
	require.Equal(t, 2, calls, "with the cache disabled every request materializes from source")

	content, state, err := Materialize(cache, identity, func() (string, error) { return "unused", nil })
	require.NoError(t, err)
	require.Equal(t, "cached", content)
	require.Equal(t, CacheHit, state, "re-enabling the cache finds the retained entry")
}

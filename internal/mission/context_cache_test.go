package mission

import (
	"errors"
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

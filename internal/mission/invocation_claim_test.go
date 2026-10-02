package mission

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestClaimTargetIsExclusiveUntilReleased(t *testing.T) {
	store := NewInvocationStore(t.TempDir())

	release, err := store.ClaimTarget("m1", "ranger", "discovery")
	require.NoError(t, err)
	_, err = store.ClaimTarget("m1", "ranger", "discovery")
	require.ErrorContains(t, err, "invocation_in_progress")

	other, err := store.ClaimTarget("m2", "ranger", "discovery")
	require.NoError(t, err, "a different mission target is independent")
	other()

	release()
	release()
	again, err := store.ClaimTarget("m1", "ranger", "discovery")
	require.NoError(t, err)
	again()
}

func TestClaimTargetReclaimsAnOldLeaseWhoseOwnerIsGone(t *testing.T) {
	store := NewInvocationStore(t.TempDir())
	_, err := store.ClaimTarget("m1", "ranger", "discovery")
	require.NoError(t, err)
	path, err := store.targetLeasePath("m1", "ranger", "discovery")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte("2147483646"), 0o600))
	old := time.Now().Add(-2 * invocationClaimTTL)
	require.NoError(t, os.Chtimes(path, old, old))

	release, err := store.ClaimTarget("m1", "ranger", "discovery")

	require.NoError(t, err)
	release()
}

func TestClaimTargetKeepsAnOldLeaseWhoseOwnerIsAlive(t *testing.T) {
	store := NewInvocationStore(t.TempDir())
	_, err := store.ClaimTarget("m1", "ranger", "discovery")
	require.NoError(t, err)
	path, err := store.targetLeasePath("m1", "ranger", "discovery")
	require.NoError(t, err)
	old := time.Now().Add(-2 * invocationClaimTTL)
	require.NoError(t, os.Chtimes(path, old, old))

	_, err = store.ClaimTarget("m1", "ranger", "discovery")

	require.ErrorContains(t, err, "invocation_in_progress")
}

func TestClaimTargetRejectsAnIncompleteTarget(t *testing.T) {
	_, err := NewInvocationStore(t.TempDir()).ClaimTarget("m1", "", "discovery")
	require.ErrorContains(t, err, "required")
}

func TestClaimTargetAllowsExactlyOneConcurrentWinner(t *testing.T) {
	store := NewInvocationStore(t.TempDir())
	var wins int
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := store.ClaimTarget("m1", "ranger", "discovery"); err == nil {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	require.Equal(t, 1, wins)
}

func TestPutPrunesRecordsConsumedOrExpiredBeyondRetention(t *testing.T) {
	root := t.TempDir()
	store := NewInvocationStore(root)
	now := time.Now().UTC()
	old := now.Add(-2 * invocationRetention)
	put := func(id string, expires time.Time, consumedAt *time.Time) {
		rec := domain.MissionInvocationRecord{Request: validMissionInvocationRequest(), CreatedAt: now}
		rec.Request.RequestID = id
		rec.ExpiresAt, rec.Consumed, rec.ConsumedAt = expires, consumedAt != nil, consumedAt
		require.NoError(t, store.Put(rec))
	}
	put("inv_expiredold", old, nil)
	put("inv_consumedold", now.Add(time.Hour), &old)
	put("inv_fresh0001", now.Add(time.Hour), nil)

	trigger := domain.MissionInvocationRecord{Request: validMissionInvocationRequest(), CreatedAt: now, ExpiresAt: now.Add(time.Hour)}
	trigger.Request.RequestID = "inv_trigger01"
	require.NoError(t, store.Put(trigger))

	require.NoFileExists(t, filepath.Join(root, "missions", "invocations", "inv_expiredold.json"))
	require.NoFileExists(t, filepath.Join(root, "missions", "invocations", "inv_consumedold.json"))
	require.FileExists(t, filepath.Join(root, "missions", "invocations", "inv_fresh0001.json"))
}

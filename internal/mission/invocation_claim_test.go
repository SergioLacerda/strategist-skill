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

func TestClaimIsExclusiveUntilReleased(t *testing.T) {
	store := NewInvocationStore(t.TempDir())

	release, err := store.Claim("inv_12345678")
	require.NoError(t, err)
	_, err = store.Claim("inv_12345678")
	require.ErrorContains(t, err, "invocation_in_progress")

	release()
	release()
	again, err := store.Claim("inv_12345678")
	require.NoError(t, err)
	again()
}

func TestClaimReclaimsAStaleMarker(t *testing.T) {
	store := NewInvocationStore(t.TempDir())
	_, err := store.Claim("inv_12345678")
	require.NoError(t, err)
	path, err := store.claimPath("inv_12345678")
	require.NoError(t, err)
	old := time.Now().Add(-2 * invocationClaimTTL)
	require.NoError(t, os.Chtimes(path, old, old))

	release, err := store.Claim("inv_12345678")

	require.NoError(t, err)
	release()
}

func TestClaimRejectsMalformedRequestID(t *testing.T) {
	_, err := NewInvocationStore(t.TempDir()).Claim("../escape")
	require.ErrorContains(t, err, "malformed request_id")
}

func TestClaimAllowsExactlyOneConcurrentWinner(t *testing.T) {
	store := NewInvocationStore(t.TempDir())
	var wins int
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := store.Claim("inv_12345678"); err == nil {
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

package mission

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func listRequest(id, missionID string) domain.MissionInvocationRequest {
	request := validMissionInvocationRequest()
	request.RequestID = id
	request.MissionID = missionID
	request.Payload = "payload-secret-value"
	request.Nonce = "nonce-secret-value"
	request.Input = map[string]any{"request_context": "context-secret-value"}
	return request
}

func TestInvocationStoreListSummarizesEffectiveStatesInCreationOrder(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	clock := now
	store := InvocationStore{Root: t.TempDir(), Clock: func() time.Time { return clock }}

	require.NoError(t, store.Put(domain.MissionInvocationRecord{Request: listRequest("inv_expired01", "m-a"), CreatedAt: now, ExpiresAt: now.Add(time.Hour)}))
	require.NoError(t, store.Put(domain.MissionInvocationRecord{Request: listRequest("inv_done0001", "m-a"), CreatedAt: now.Add(time.Minute), ExpiresAt: now.Add(2 * time.Hour)}))
	require.NoError(t, store.BeginProcessing("inv_done0001", "t.md", "sha256:a"))
	require.NoError(t, store.Complete("inv_done0001", "sha256:a"))
	require.NoError(t, store.Put(domain.MissionInvocationRecord{Request: listRequest("inv_live0001", "m-a"), CreatedAt: now.Add(2 * time.Hour), ExpiresAt: now.Add(3 * time.Hour)}))
	require.NoError(t, store.Put(domain.MissionInvocationRecord{Request: listRequest("inv_other001", "m-b"), CreatedAt: now.Add(30 * time.Minute), ExpiresAt: now.Add(90 * time.Minute)}))
	clock = now.Add(2*time.Hour + 10*time.Minute)

	listing, err := store.List("m-a")
	require.NoError(t, err)
	require.Empty(t, listing.Skipped)
	require.Len(t, listing.Requests, 3, "the other mission is filtered out")

	require.Equal(t, []string{"inv_expired01", "inv_done0001", "inv_live0001"}, summaryIDs(listing.Requests), "creation order")
	byID := map[string]domain.MissionInvocationSummary{}
	for _, summary := range listing.Requests {
		byID[summary.RequestID] = summary
	}
	require.Equal(t, domain.InvocationStatePending, byID["inv_expired01"].State)
	require.True(t, byID["inv_expired01"].Expired, "a pending request past expires_at is flagged, not hidden")
	require.Equal(t, domain.InvocationStateCompleted, byID["inv_done0001"].State)
	require.False(t, byID["inv_done0001"].Expired, "a completed request is never expired")
	require.Equal(t, domain.InvocationStatePending, byID["inv_live0001"].State)
	require.False(t, byID["inv_live0001"].Expired)
	require.Equal(t, "ranger", byID["inv_live0001"].Role)
	require.Equal(t, "discovery", byID["inv_live0001"].Slot)
	require.Equal(t, "m-a", byID["inv_live0001"].MissionID)
}

func TestInvocationStoreListWithoutMissionReturnsEveryMission(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	store := InvocationStore{Root: t.TempDir(), Clock: func() time.Time { return now }}
	require.NoError(t, store.Put(domain.MissionInvocationRecord{Request: listRequest("inv_aaaaaaaa", "m-a"), CreatedAt: now, ExpiresAt: now.Add(time.Hour)}))
	require.NoError(t, store.Put(domain.MissionInvocationRecord{Request: listRequest("inv_bbbbbbbb", "m-b"), CreatedAt: now.Add(time.Second), ExpiresAt: now.Add(time.Hour)}))

	listing, err := store.List("")
	require.NoError(t, err)
	require.Equal(t, []string{"inv_aaaaaaaa", "inv_bbbbbbbb"}, summaryIDs(listing.Requests))
}

func TestInvocationStoreListNeverExposesPayloadInputOrNonce(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	store := InvocationStore{Root: t.TempDir(), Clock: func() time.Time { return now }}
	require.NoError(t, store.Put(domain.MissionInvocationRecord{Request: listRequest("inv_secret001", "m-a"), CreatedAt: now, ExpiresAt: now.Add(time.Hour)}))

	listing, err := store.List("m-a")
	require.NoError(t, err)
	encoded, err := json.Marshal(listing)
	require.NoError(t, err)
	for _, secret := range []string{"payload-secret-value", "nonce-secret-value", "context-secret-value", "payload", "nonce"} {
		require.NotContains(t, string(encoded), secret)
	}
}

func TestInvocationStoreListWithoutDirectoryIsEmptyNotAnError(t *testing.T) {
	store := InvocationStore{Root: t.TempDir(), Clock: time.Now}

	listing, err := store.List("m-a")
	require.NoError(t, err)
	require.Empty(t, listing.Requests)
	require.Empty(t, listing.Skipped)
}

func TestInvocationStoreListReportsUnreadableRecordsByNameOnly(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	store := InvocationStore{Root: t.TempDir(), Clock: func() time.Time { return now }}
	require.NoError(t, store.Put(domain.MissionInvocationRecord{Request: listRequest("inv_goodgood", "m-a"), CreatedAt: now, ExpiresAt: now.Add(time.Hour)}))
	dir := filepath.Join(store.Root, "missions", "invocations")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "inv_corrupt01.json"), []byte("{not json secret-body"), 0o600))

	listing, err := store.List("m-a")
	require.NoError(t, err)
	require.Equal(t, []string{"inv_goodgood"}, summaryIDs(listing.Requests))
	require.Equal(t, []string{"inv_corrupt01.json"}, listing.Skipped, "the file name is reported, never its content")
}

func summaryIDs(summaries []domain.MissionInvocationSummary) []string {
	ids := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		ids = append(ids, summary.RequestID)
	}
	return ids
}

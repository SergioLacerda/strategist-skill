package mission

import (
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestInvocationStorePersistsAndConsumesOnce(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	store := InvocationStore{Root: t.TempDir(), Clock: func() time.Time { return now }}
	record := domain.MissionInvocationRecord{Request: validMissionInvocationRequest(), CreatedAt: now, ExpiresAt: now.Add(time.Minute)}
	require.NoError(t, store.Put(record))
	got, err := store.Get(record.Request.RequestID)
	require.NoError(t, err)
	require.Equal(t, record.Request, got.Request)
	require.NoError(t, store.Consume(record.Request.RequestID))
	require.ErrorContains(t, mustGet(store, record.Request.RequestID), "invocation_replay")
}

func TestInvocationStoreRejectsExpiredRequest(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	store := InvocationStore{Root: t.TempDir(), Clock: func() time.Time { return now.Add(time.Minute) }}
	record := domain.MissionInvocationRecord{Request: validMissionInvocationRequest(), CreatedAt: now, ExpiresAt: now.Add(30 * time.Second)}
	require.NoError(t, store.Put(record))
	_, err := store.Get(record.Request.RequestID)
	require.ErrorContains(t, err, "invocation_request_expired")
}

func validMissionInvocationRequest() domain.MissionInvocationRequest {
	return domain.MissionInvocationRequest{Protocol: domain.MissionInvocationProtocolVersion, RequestID: "inv_12345678", MissionID: "mission", Role: "ranger", Slot: "discovery", Weapon: domain.MissionWeaponIdentity{ID: "brainstorming", Version: "1.0.0", Digest: "sha256:weapon"}, BindingDigest: "sha256:binding", SourceDigest: "sha256:source", ExecutionMode: "prompt_bridge", Entrypoint: "discover", Payload: "payload"}
}

func mustGet(store InvocationStore, id string) error {
	_, err := store.Get(id)
	return err
}

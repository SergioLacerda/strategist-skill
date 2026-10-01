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
	require.NoError(t, store.BeginProcessing(record.Request.RequestID, ".analysis/pending/m.md", "sha256:a"))
	require.NoError(t, store.Complete(record.Request.RequestID, "sha256:a"))
	require.ErrorContains(t, mustGet(store, record.Request.RequestID), "invocation_replay")
}

func TestInvocationStoreRejectsInvalidAndBackwardTransitions(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	store := InvocationStore{Root: t.TempDir(), Clock: func() time.Time { return now }}
	id := validMissionInvocationRequest().RequestID
	require.NoError(t, store.Put(domain.MissionInvocationRecord{Request: validMissionInvocationRequest(), CreatedAt: now, ExpiresAt: now.Add(time.Minute)}))

	require.ErrorContains(t, store.Complete(id, "sha256:a"), "invocation_state_invalid", "pending cannot skip processing")
	require.NoError(t, store.BeginProcessing(id, "t.md", "sha256:a"))
	require.NoError(t, store.BeginProcessing(id, "t.md", "sha256:a"), "processing may re-enter for recovery")
	require.ErrorContains(t, store.BeginProcessing(id, "t.md", "sha256:other"), "invocation_digest_mismatch")
	require.ErrorContains(t, store.Complete(id, "sha256:other"), "invocation_digest_mismatch")
	require.NoError(t, store.Complete(id, "sha256:a"))
	require.ErrorContains(t, store.BeginProcessing(id, "t.md", "sha256:a"), "invocation_replay")
	require.ErrorContains(t, store.Complete(id, "sha256:a"), "invocation_replay")
}

func TestInvocationStoreProcessingSurvivesExpiryAndCompletionCompactsTheRecord(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	clock := now
	store := InvocationStore{Root: t.TempDir(), Clock: func() time.Time { return clock }}
	request := validMissionInvocationRequest()
	request.Input = map[string]any{"request_context": "private"}
	require.NoError(t, store.Put(domain.MissionInvocationRecord{Request: request, CreatedAt: now, ExpiresAt: now.Add(time.Minute)}))
	require.NoError(t, store.BeginProcessing(request.RequestID, "t.md", "sha256:a"))

	clock = now.Add(time.Hour)
	got, err := store.Get(request.RequestID)
	require.NoError(t, err, "recovery must not be expired out from under a committed completion")
	require.Equal(t, domain.InvocationStateProcessing, got.EffectiveState())

	require.NoError(t, store.Complete(request.RequestID, "sha256:a"))
	done, err := store.load(request.RequestID)
	require.NoError(t, err)
	require.Empty(t, done.Request.Payload)
	require.Empty(t, done.Request.Input)
	require.Equal(t, request.RequestID, done.Request.RequestID)
	require.Equal(t, "sha256:a", done.ArtifactDigest)
	require.NotNil(t, done.CompletedAt)
}

func TestInvocationStoreMapsPreStateRecords(t *testing.T) {
	require.Equal(t, domain.InvocationStatePending, domain.MissionInvocationRecord{}.EffectiveState())
	require.Equal(t, domain.InvocationStateCompleted, domain.MissionInvocationRecord{Consumed: true}.EffectiveState())
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

func TestCommitExecutionAdapterIsOneWayAndPendingOnly(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	store := InvocationStore{Root: t.TempDir(), Clock: func() time.Time { return now }}
	record := domain.MissionInvocationRecord{Request: validMissionInvocationRequest(), CreatedAt: now, ExpiresAt: now.Add(time.Minute), ExecutionAdapter: domain.ExecutionAdapterCurrentHost}
	id := record.Request.RequestID
	require.NoError(t, store.Put(record))

	require.ErrorContains(t, store.CommitExecutionAdapter(id, domain.ExecutionAdapterCurrentHost, "p"), "invocation_adapter_unknown", "only child modes are committed")
	require.ErrorContains(t, store.CommitExecutionAdapter(id, "bogus", "p"), "invocation_adapter_unknown")
	require.ErrorContains(t, store.CommitExecutionAdapter(id, domain.ExecutionAdapterCodexChild, ""), "invocation_adapter_unknown")
	require.NoError(t, store.CommitExecutionAdapter(id, domain.ExecutionAdapterCodexChild, "p1"))
	require.NoError(t, store.CommitExecutionAdapter(id, domain.ExecutionAdapterCodexChild, "p1"), "same commitment is idempotent")
	require.ErrorContains(t, store.CommitExecutionAdapter(id, domain.ExecutionAdapterClaudeChild, "p2"), "invocation_adapter_mismatch")
	require.ErrorContains(t, store.CommitExecutionAdapter(id, domain.ExecutionAdapterCodexChild, "other"), "invocation_adapter_mismatch")

	require.NoError(t, store.BeginProcessing(id, "t.md", "sha256:a"))
	require.ErrorContains(t, store.CommitExecutionAdapter(id, domain.ExecutionAdapterCodexChild, "p1"), "invocation_state_invalid")
	got, err := store.Get(id)
	require.NoError(t, err)
	require.Equal(t, domain.ExecutionAdapterCodexChild, got.ExecutionAdapter, "the commitment survives the transaction")
	require.Equal(t, "p1", got.ChildPolicyID)
}

func TestCommitExecutionAdapterNeverUpgradesAPreFieldRecord(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	store := InvocationStore{Root: t.TempDir(), Clock: func() time.Time { return now }}
	record := domain.MissionInvocationRecord{Request: validMissionInvocationRequest(), CreatedAt: now, ExpiresAt: now.Add(time.Minute)}
	require.NoError(t, store.Put(record))

	require.ErrorContains(t, store.CommitExecutionAdapter(record.Request.RequestID, domain.ExecutionAdapterCodexChild, "p"), "invocation_adapter_mismatch")
}

func TestPutRejectsAnUncommittableAdapter(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	store := InvocationStore{Root: t.TempDir(), Clock: func() time.Time { return now }}
	for _, adapter := range []domain.MissionExecutionAdapter{"bogus", domain.ExecutionAdapterCurrentHostUnverified} {
		record := domain.MissionInvocationRecord{Request: validMissionInvocationRequest(), CreatedAt: now, ExpiresAt: now.Add(time.Minute), ExecutionAdapter: adapter}
		require.ErrorContains(t, store.Put(record), "invocation_adapter_unknown")
	}
}

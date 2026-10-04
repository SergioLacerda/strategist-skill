package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/stretchr/testify/require"
)

func TestListMissionInvocationRequestsUsesTheStoreWithoutExposingSecrets(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	request := domain.MissionInvocationRequest{
		Protocol: domain.MissionInvocationProtocolVersion, RequestID: "inv_listwire01", MissionID: "m-list", Role: "ranger", Slot: "discovery",
		ExecutionMode: "prompt_bridge", Entrypoint: "discover", Payload: "payload-secret-value", Nonce: "nonce-secret-value",
		Weapon: domain.MissionWeaponIdentity{ID: "brainstorming", Version: "1.0.0", Digest: "sha256:a"}, BindingDigest: "sha256:b", SourceDigest: "sha256:c",
	}
	require.NoError(t, missionruntime.NewInvocationStore(root).Put(domain.MissionInvocationRecord{Request: request, CreatedAt: now, ExpiresAt: now.Add(time.Hour)}))

	listing, err := listMissionInvocationRequests(root, "m-list")
	require.NoError(t, err)
	require.Len(t, listing.Requests, 1)
	require.Equal(t, "inv_listwire01", listing.Requests[0].RequestID)
	require.False(t, listing.Requests[0].Expired)
	encoded, err := json.Marshal(listing)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "secret-value")

	other, err := listMissionInvocationRequests(root, "another-mission")
	require.NoError(t, err)
	require.Empty(t, other.Requests)
}

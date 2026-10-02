package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestMissionInvocationRequestValidateRequiresEmbeddedIdentity(t *testing.T) {
	req := domain.MissionInvocationRequest{Protocol: domain.MissionInvocationProtocolVersion, RequestID: "inv_12345678", MissionID: "mission", Role: "ranger", Slot: "discovery", Weapon: domain.MissionWeaponIdentity{ID: "brainstorming", Version: "1.0.0", Digest: "sha256:weapon"}, BindingDigest: "sha256:binding", SourceDigest: "sha256:source", ExecutionMode: "prompt_bridge", Entrypoint: "discover", Payload: "payload"}
	require.NoError(t, req.Validate())
	req.Payload = ""
	require.ErrorContains(t, req.Validate(), "payload is required")
}

func TestMissionInvocationCompletionRejectsEmptyResult(t *testing.T) {
	require.ErrorContains(t, (domain.MissionInvocationCompletion{RequestID: "inv_12345678"}).Validate(), "result is required")
}

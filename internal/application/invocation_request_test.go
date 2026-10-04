package application

import (
	"context"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestNewInvocationRequestBuildsContractsAndImmutableIdentity(t *testing.T) {
	request, issued, err := NewInvocationRequest(t.Context(), InvocationRequestInput{
		BasePath: "analysis", MissionID: "m1", Role: "ranger", Slot: "discovery",
		RequestContext: "evaluate", Binding: domain.RoleWeaponBinding{
			BindingDigest: "sha256:binding", SourceDigest: "sha256:source", ExecutionMode: "prompt", Entrypoint: "run",
		}, Weapon: domain.CompiledWeapon{ID: "brainstorming", Version: "1.0.0", Digest: "sha256:weapon"},
		Payload: []byte("instructions"), SourceDigest: "sha256:source",
		ExecutionContract: "execute once", OutputContract: "return handoff",
	})

	require.NoError(t, err)
	require.False(t, issued.IsZero())
	require.NoError(t, request.Validate())
	require.Equal(t, "execute once", request.Input["execution_contract"])
	require.Equal(t, "return handoff", request.Input["output_contract"])
	require.Equal(t, "evaluate", request.Input["request_context"])
	require.NotEmpty(t, request.RequestID)
	require.NotEmpty(t, request.Nonce)
}

func TestNewInvocationRequestHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, _, err := NewInvocationRequest(ctx, InvocationRequestInput{})
	require.ErrorContains(t, err, "context")
}

func TestInvocationValidationRejectsUnsupportedBoundaries(t *testing.T) {
	require.ErrorContains(t, ValidateEmbeddedInvocationBinding(domain.RoleWeaponBinding{Mode: domain.SlotBindingModeCustom}), "only supports Ranked")
	require.ErrorContains(t, ValidateEmbeddedInvocationBinding(domain.RoleWeaponBinding{Mode: domain.SlotBindingModeRanked, RuntimeKind: domain.RankedRuntimeOpenSpecRoot}), "normalize-openspec")
	require.ErrorContains(t, ValidateEmbeddedInvocationBinding(domain.RoleWeaponBinding{Mode: domain.SlotBindingModeRanked, RuntimeKind: domain.RankedRuntimeHost}), "runtime kind")

	record := domain.MissionInvocationRecord{Request: domain.MissionInvocationRequest{RequestID: "req", Role: "ranger", Slot: string(domain.SlotDiscovery)}}
	require.ErrorContains(t, ValidateInvocationCompletion(record, domain.MissionInvocationCompletion{RequestID: "other", Result: "ok"}), "request_id")
	require.NoError(t, ValidateInvocationCompletion(record, domain.MissionInvocationCompletion{RequestID: "req", Result: "ok"}))
}

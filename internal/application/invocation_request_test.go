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

func TestNewInvocationRequestCarriesTypedExecutionTargetsForSniper(t *testing.T) {
	target := domain.ExecutionTarget{
		Path: "docs/adr/m1-adr.md", Kind: domain.ExecutionTargetKindADR,
		Origin: domain.ExecutionTargetOriginAcceptedSideQuest, SideQuestID: "OA-ADR-m1",
	}
	request, _, err := NewInvocationRequest(t.Context(), InvocationRequestInput{
		BasePath: "analysis", MissionID: "m1", Role: "sniper", Slot: string(domain.SlotExecution),
		Binding:                   domain.RoleWeaponBinding{BindingDigest: "sha256:binding", ExecutionMode: "prompt", Entrypoint: "run"},
		Weapon:                    domain.CompiledWeapon{ID: "writing-plans", Version: "1.0.0", Digest: "sha256:weapon"},
		ApprovalGatePackageDigest: "sha256:package", ExecutionTargets: []domain.ExecutionTarget{target},
	})
	require.NoError(t, err)
	require.Equal(t, []domain.ExecutionTarget{target}, request.Input["execution_targets"])
}

func TestNewInvocationRequestCarriesDeferredNoticesInTheSnapshot(t *testing.T) {
	snapshot := domain.ExecutionSnapshot{
		Version: domain.ExecutionSnapshotVersion, MissionID: "m1",
		Groups: []domain.ExecutionGroup{{
			ID: "documentation-m1", Resolver: domain.ResolverKey{Kind: "documentation", Origin: domain.ExecutionTargetOriginRefinedPackage},
			Kind: "documentation", Origin: domain.ExecutionTargetOriginRefinedPackage, CompletionPolicy: domain.ExecutionCompletionAllRequired,
			Artifacts: []domain.ExecutionArtifact{{ID: "a1", Role: domain.ExecutionArtifactRolePrimary, Path: "docs/a.md", Kind: "documentation"}},
		}},
		Deferred: []domain.DeferredIntent{{SideQuestID: "OA-CHEST-m1", Kind: "oa_treasure_chest", Origin: domain.ExecutionTargetOriginAcceptedSideQuest, Reason: domain.DeferredReasonNoExecutionResolver}},
	}
	request, _, err := NewInvocationRequest(t.Context(), InvocationRequestInput{
		BasePath: "analysis", MissionID: "m1", Role: "sniper", Slot: string(domain.SlotExecution),
		Binding:                   domain.RoleWeaponBinding{BindingDigest: "sha256:binding", ExecutionMode: "prompt", Entrypoint: "run"},
		Weapon:                    domain.CompiledWeapon{ID: "sniper", Version: "0.0.0", Digest: "sha256:weapon"},
		ApprovalGatePackageDigest: "sha256:package", ExecutionSnapshot: &snapshot,
	})
	require.NoError(t, err)
	carried, ok := request.Input["execution_snapshot"].(domain.ExecutionSnapshot)
	require.True(t, ok)
	require.Equal(t, snapshot.Deferred, carried.Deferred)
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

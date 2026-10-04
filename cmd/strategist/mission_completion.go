package main

import (
	"context"
	"fmt"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/SergioLacerda/strategist-skill/internal/provider"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// completeMissionInvocation publishes the pending Ranger artifact for one
// request. The publication target (mission, Role, slot) is leased before any
// state or artifact is inspected, so two request IDs can never race over it.
func completeMissionInvocation(ctx context.Context, input missionadapter.InvocationCompleteInput) (domain.MissionInvocationOutcome, error) {
	store := missionruntime.NewInvocationStore(input.Root)
	record, release, err := claimCompletableInvocation(store, input)
	if err != nil {
		return domain.MissionInvocationOutcome{}, err
	}
	defer release()
	if record.Request.Role == "sniper" && record.Request.Slot == string(domain.SlotExecution) {
		return completeSniperInvocation(store, input, record)
	}
	if input.Sink == nil {
		return domain.MissionInvocationOutcome{}, fmt.Errorf("invocation_telemetry_unavailable: mission completion requires a configured telemetry sink")
	}
	artifactPath, err := publishDiscoveryArtifact(ctx, store, input, record)
	if err != nil {
		return domain.MissionInvocationOutcome{}, err
	}
	if err := recordRangerHandoffAfterPublish(ctx, input, record); err != nil {
		return domain.MissionInvocationOutcome{}, err
	}
	return domain.MissionInvocationOutcome{RequestID: input.RequestID, MissionID: record.Request.MissionID, Status: "normalized", ArtifactPath: artifactPath, BindingDigest: record.Request.BindingDigest, SourceDigest: record.Request.SourceDigest}, nil
}

// claimCompletableInvocation leases the publication target and re-reads the
// request under the lease (another completion may have finished meanwhile),
// then verifies the adapter and that the request is still current. The caller
// releases the lease; on any error it is already released.
func claimCompletableInvocation(store missionruntime.InvocationStore, input missionadapter.InvocationCompleteInput) (domain.MissionInvocationRecord, func(), error) {
	issued, err := store.Get(input.RequestID)
	if err != nil {
		return domain.MissionInvocationRecord{}, nil, fmt.Errorf("load mission invocation: %w", err)
	}
	release, err := store.ClaimTarget(issued.Request.MissionID, issued.Request.Role, issued.Request.Slot)
	if err != nil {
		return domain.MissionInvocationRecord{}, nil, fmt.Errorf("claim mission invocation: %w", err)
	}
	record, err := verifyClaimedInvocation(store, input)
	if err != nil {
		release()
		return domain.MissionInvocationRecord{}, nil, err
	}
	return record, release, nil
}

func verifyClaimedInvocation(store missionruntime.InvocationStore, input missionadapter.InvocationCompleteInput) (domain.MissionInvocationRecord, error) {
	record, err := loadCompletableInvocation(store, input)
	if err != nil {
		return domain.MissionInvocationRecord{}, err
	}
	if err := verifyExecutionAdapter(record, input); err != nil {
		return domain.MissionInvocationRecord{}, err
	}
	if err := verifyInvocationStillCurrent(input.Root, record); err != nil {
		return domain.MissionInvocationRecord{}, err
	}
	return record, nil
}

// recordRangerHandoffAfterPublish records the Ranger-to-Archivist handoff once
// a discovery artifact is published; other roles and slots have none.
func recordRangerHandoffAfterPublish(ctx context.Context, input missionadapter.InvocationCompleteInput, record domain.MissionInvocationRecord) error {
	if record.Request.Role != "ranger" || record.Request.Slot != string(domain.SlotDiscovery) {
		return nil
	}
	delegate := rangerDelegate(ctx, input.Root, input.BasePath, record.Request.MissionID)
	if err := missionruntime.EnsureRangerToArchivistOutcomeWithDelegation(ctx, input.Root, input.BasePath, record.Request.MissionID, input.Sink, record.Request.MissionID, delegate); err != nil {
		return fmt.Errorf("ranger-to-archivist handoff: %w", err)
	}
	return nil
}

// verifyInvocationStillCurrent rejects a request whose mission left its slot
// or whose Ranked binding changed after the request was issued.
func verifyInvocationStillCurrent(root string, record domain.MissionInvocationRecord) error {
	expected := map[string]domain.PipelinePhase{
		string(domain.SlotDiscovery):  domain.PhaseDiscovery,
		string(domain.SlotRefinement): domain.PhaseRefinement,
		string(domain.SlotExecution):  domain.PhaseExecution,
	}[record.Request.Slot]
	phase, found := missionruntime.ReadMissionPhase(root, record.Request.MissionID)
	if expected == "" || !found || phase != expected {
		return fmt.Errorf("invocation_phase_mismatch: mission %q is not in %s (found=%t, phase=%q)", record.Request.MissionID, expected, found, phase)
	}
	binding, _, _, _, err := resolveMissionInvocationWeapon(missionadapter.InvocationBuildInput{
		Root: root, MissionID: record.Request.MissionID, Role: record.Request.Role, Slot: record.Request.Slot,
	})
	if err != nil {
		return err
	}
	if binding.BindingDigest != record.Request.BindingDigest {
		return fmt.Errorf("invocation_binding_mismatch: the active binding changed since the request was issued")
	}
	return nil
}

func loadCompletableInvocation(store missionruntime.InvocationStore, input missionadapter.InvocationCompleteInput) (domain.MissionInvocationRecord, error) {
	record, err := store.Get(input.RequestID)
	if err != nil {
		return domain.MissionInvocationRecord{}, fmt.Errorf("load mission invocation: %w", err)
	}
	if err := application.ValidateInvocationCompletion(record, input.Completion); err != nil {
		return domain.MissionInvocationRecord{}, wrapMissionError(err)
	}
	return record, nil
}

func normalizeDiscoveryInvocation(ctx context.Context, sink telemetry.EventSink, record domain.MissionInvocationRecord, artifactPath, result string) (provider.NormalizedDiscoveryArtifact, error) {
	request := provider.DiscoveryWeaponRequest{
		MissionID: record.Request.MissionID, Role: record.Request.Role, Slot: record.Request.Slot,
		ProviderID: record.Request.Weapon.ID, RuntimeKind: domain.RankedRuntimeEmbedded,
		BindingDigest: record.Request.BindingDigest, SourceDigest: record.Request.SourceDigest,
		Entrypoint: record.Request.Entrypoint, ArtifactPath: artifactPath, InvocationNonce: record.Request.Nonce,
		ExecutionAdapter: record.EffectiveAdapter(), ChildPolicyID: record.ChildPolicyID,
		InvocationRequestID: record.Request.RequestID,
	}
	artifact, err := provider.InvokeAndNormalizeDiscoveryWithTelemetry(ctx, request, func(context.Context, provider.DiscoveryWeaponRequest) (provider.DiscoveryWeaponResponse, error) {
		return provider.DiscoveryWeaponResponse{
			ProviderID: record.Request.Weapon.ID, InvocationEvidence: "embedded_prompt_bridge",
			Artifact: []byte(result), EmbeddedInvocationReceipt: connectors.EmbeddedInvocationReceipt{
				SchemaVersion: connectors.EmbeddedInvocationReceiptSchemaVersion, MissionID: record.Request.MissionID,
				Role: record.Request.Role, Slot: record.Request.Slot, WeaponID: record.Request.Weapon.ID,
				Entrypoint: record.Request.Entrypoint, BindingDigest: record.Request.BindingDigest,
				SourceDigest: record.Request.SourceDigest, RequestID: record.Request.RequestID, Nonce: record.Request.Nonce, ExecutionAdapter: record.EffectiveAdapter(), ChildPolicyID: record.ChildPolicyID, IssuedAt: record.CreatedAt,
			},
		}, nil
	}, sink, "")
	if err != nil {
		return provider.NormalizedDiscoveryArtifact{}, fmt.Errorf("normalize discovery completion: %w", err)
	}
	return artifact, nil
}

func discoveryArtifactPaths(root, basePath, missionID string) (string, string, error) {
	relative, absolute, err := missionruntime.DiscoveryArtifactPaths(root, basePath, missionID)
	return relative, absolute, wrapMissionError(err)
}

func writeMissionArtifact(path string, content []byte) error {
	return wrapMissionError(missionruntime.WriteDiscoveryArtifact(path, content))
}

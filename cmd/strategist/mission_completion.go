package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
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
	if err := missionruntime.EnsureRangerToArchivistOutcomeWithTelemetry(ctx, input.Root, input.BasePath, record.Request.MissionID, input.Sink, record.Request.MissionID); err != nil {
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
	if err := input.Completion.Validate(); err != nil {
		return domain.MissionInvocationRecord{}, fmt.Errorf("validate mission completion: %w", err)
	}
	if input.Completion.RequestID != record.Request.RequestID {
		return domain.MissionInvocationRecord{}, fmt.Errorf("invocation_binding_mismatch: completion request_id does not match pending request")
	}
	registered := (record.Request.Role == "ranger" && record.Request.Slot == string(domain.SlotDiscovery)) ||
		(record.Request.Role == "sniper" && record.Request.Slot == string(domain.SlotExecution))
	if !registered {
		return domain.MissionInvocationRecord{}, fmt.Errorf("role_invocation_failed: completion normalization is not registered for %s/%s", record.Request.Role, record.Request.Slot)
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
	workspace := filepath.Dir(root)
	absolute := filepath.Join(basePath, "pending", missionID+"-analysis.md")
	relative, err := filepath.Rel(workspace, absolute)
	if err != nil || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("role_invocation_failed: discovery artifact path escapes workspace")
	}
	return filepath.ToSlash(relative), absolute, nil
}

func writeMissionArtifact(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("write normalized artifact: create directory: %w", err)
	}
	if err := missionruntime.WriteFileAtomic(path, content, 0o644); err != nil {
		return fmt.Errorf("write normalized artifact: %w", err)
	}
	return nil
}

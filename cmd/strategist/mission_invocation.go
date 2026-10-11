package main

import (
	"context"
	"fmt"
	"time"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/provider"
)

// invocationLifetime covers a manual host run (the agent works between invoke
// and complete); host-bridge runs are bounded by the shorter hostBridgeTimeout.
const invocationLifetime = time.Hour

func buildMissionInvocation(ctx context.Context, input missionadapter.InvocationBuildInput) (domain.MissionInvocationRequest, error) {
	if err := ctx.Err(); err != nil {
		return domain.MissionInvocationRequest{}, fmt.Errorf("build mission invocation: context: %w", err)
	}
	binding, weapon, payload, sourceDigest, err := resolveMissionInvocationWeapon(input)
	if err != nil {
		return domain.MissionInvocationRequest{}, err
	}
	executionTargets, err := loadSniperExecutionTargets(input)
	if err != nil {
		return domain.MissionInvocationRequest{}, err
	}
	request, now, err := newMissionInvocationRequest(ctx, input, binding, weapon, payload, sourceDigest, executionTargets)
	if err != nil {
		return domain.MissionInvocationRequest{}, err
	}
	if err := authorizeArchivistEntry(input); err != nil {
		return domain.MissionInvocationRequest{}, err
	}
	if err := authorizeSniperEntry(input); err != nil {
		return domain.MissionInvocationRequest{}, err
	}
	store := missionruntime.NewInvocationStore(input.Root)
	if err := store.Put(domain.MissionInvocationRecord{Request: request, CreatedAt: now, ExpiresAt: now.Add(invocationLifetime), ExecutionAdapter: domain.ExecutionAdapterCurrentHost}); err != nil {
		return domain.MissionInvocationRequest{}, fmt.Errorf("persist mission invocation: %w", err)
	}
	return request, nil
}

func loadSniperExecutionTargets(input missionadapter.InvocationBuildInput) ([]domain.ExecutionTarget, error) {
	if input.Role != "sniper" || input.Slot != string(domain.SlotExecution) {
		return nil, nil
	}
	targets, err := missionadapter.LoadExecutionTargetSnapshot(input.Root, input.BasePath, input.MissionID)
	if err != nil {
		return nil, fmt.Errorf("load Sniper execution targets: %w", err)
	}
	return targets, nil
}

func newMissionInvocationRequest(ctx context.Context, input missionadapter.InvocationBuildInput, binding domain.RoleWeaponBinding, weapon domain.CompiledWeapon, payload []byte, sourceDigest string, executionTargets []domain.ExecutionTarget) (domain.MissionInvocationRequest, time.Time, error) {
	approvalDigest := ""
	stageCorrelationID := ""
	executionContract, outputContract := invocationContracts(input.Role, input.Slot)
	if input.Role == "sniper" && input.Slot == string(domain.SlotExecution) {
		_, status, loadErr := loadMission(input.Root, input.MissionID)
		if loadErr != nil {
			return domain.MissionInvocationRequest{}, time.Time{}, fmt.Errorf("load Sniper mission authorization: %w", loadErr)
		}
		approvalDigest = status.ApprovalGatePackageDigest
		stageCorrelationID = criticalHitStageCorrelation(status)
	}
	request, issued, err := application.NewInvocationRequest(ctx, application.InvocationRequestInput{
		Root: input.Root, BasePath: input.BasePath, MissionID: input.MissionID, Role: input.Role, Slot: input.Slot,
		RequestContext: input.RequestContext, Binding: binding, Weapon: weapon, Payload: payload,
		SourceDigest: sourceDigest, ApprovalGatePackageDigest: approvalDigest, StageCorrelationID: stageCorrelationID,
		ExecutionContract: executionContract, OutputContract: outputContract,
		ExecutionTargets:  executionTargets,
		ExecutionSnapshot: missionExecutionSnapshot(input, executionTargets),
	})
	return request, issued, wrapMissionError(err)
}

func missionExecutionSnapshot(input missionadapter.InvocationBuildInput, executionTargets []domain.ExecutionTarget) *domain.ExecutionSnapshot {
	if input.Role != "sniper" || input.Slot != string(domain.SlotExecution) {
		return nil
	}
	snapshot, err := missionadapter.LoadExecutionSnapshot(input.Root, input.BasePath, input.MissionID)
	if err == nil && len(snapshot.Groups) > 0 {
		return &snapshot
	}
	legacy, err := domain.SnapshotFromTargets(input.MissionID, executionTargets)
	if err != nil || len(legacy.Groups) == 0 {
		return nil
	}
	return &legacy
}

// invocationContracts names the execution and output contract an embedded
// Weapon invocation carries for a Role and slot; a pair without one issues an
// envelope with neither.
func invocationContracts(role, slot string) (executionContract, outputContract string) {
	switch {
	case role == "ranger":
		return provider.DiscoveryExecutionContract, provider.DiscoveryOutputContract
	case role == "archivist" && slot == string(domain.SlotRefinement):
		return provider.ArchivistRefinementExecutionContract, provider.ArchivistRefinementOutputContract
	case role == "sniper" && slot == string(domain.SlotExecution):
		return provider.SniperExecutionContract, provider.SniperOutputContract
	}
	return "", ""
}

func criticalHitStageCorrelation(status domain.MissionEngineStatus) string {
	if status.Stage == domain.StageShort && status.StageFeat == "critical_hit" {
		return status.StageCorrelationID
	}
	return ""
}

func newInvocationID() (string, error) {
	id, err := application.NewInvocationID()
	return id, wrapMissionError(err)
}

// listMissionInvocationRequests is the read-only composition of the request listing.
func listMissionInvocationRequests(root, missionID string) (domain.MissionInvocationListing, error) {
	listing, err := missionruntime.NewInvocationStore(root).List(missionID)
	if err != nil {
		return domain.MissionInvocationListing{}, fmt.Errorf("list mission invocation requests: %w", err)
	}
	return listing, nil
}

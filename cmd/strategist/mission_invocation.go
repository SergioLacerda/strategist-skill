package main

import (
	"context"
	"fmt"
	"time"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/provider"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	telemetrysink "github.com/SergioLacerda/strategist-skill/internal/telemetry/sink"
)

// invocationLifetime covers a manual host run (the agent works between invoke
// and complete); host-bridge runs are bounded by the shorter hostBridgeTimeout.
const invocationLifetime = time.Hour

func missionInvocationDependencies() missionadapter.InvocationDependencies {
	return missionadapter.InvocationDependencies{
		RootFlag: cliutilFlagRoot(), RequireMissionID: requireMissionID,
		ResolveBasePath: resolveActiveBasePathForMission,
		LoadMission: func(root, missionID string) (domain.MissionEngineStatus, error) {
			_, status, err := loadMission(root, missionID)
			return status, err
		},
		Build: buildMissionInvocation, Complete: completeMissionInvocation, ExecuteHost: executeMissionHost,
		WriteResult: writeMissionResult, ReadCompletion: readMissionCompletion,
		TelemetrySink: selectDiscoveryTelemetrySink,
	}
}

// selectDiscoveryTelemetrySink applies the existing telemetry selection policy
// (slog or OTel, resilient per STRATEGIST_TELEMETRY_STRICT) to the environment
// read when the command runs. There is no external governance bridge in
// ordinary CLI composition, which selects no alternate sink.
func selectDiscoveryTelemetrySink() telemetry.EventSink {
	return telemetrysink.Select(telemetry.FromEnv(), nil)
}

func cliutilFlagRoot() string { return cliutil.FlagRoot }

func resolveActiveBasePathForMission(root string) (string, string, error) {
	strategistRoot, basePath, err := cliutil.ResolveActiveBasePath(root)
	if err != nil {
		return "", "", fmt.Errorf("resolve active base path: %w", err)
	}
	return strategistRoot, basePath, nil
}

func buildMissionInvocation(ctx context.Context, input missionadapter.InvocationBuildInput) (domain.MissionInvocationRequest, error) {
	if err := ctx.Err(); err != nil {
		return domain.MissionInvocationRequest{}, fmt.Errorf("build mission invocation: context: %w", err)
	}
	binding, weapon, payload, sourceDigest, err := resolveMissionInvocationWeapon(input)
	if err != nil {
		return domain.MissionInvocationRequest{}, err
	}
	request, now, err := newMissionInvocationRequest(ctx, input, binding, weapon, payload, sourceDigest)
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

func newMissionInvocationRequest(ctx context.Context, input missionadapter.InvocationBuildInput, binding domain.RoleWeaponBinding, weapon domain.CompiledWeapon, payload []byte, sourceDigest string) (domain.MissionInvocationRequest, time.Time, error) {
	approvalDigest := ""
	executionContract, outputContract := "", ""
	if input.Role == "ranger" {
		executionContract, outputContract = provider.DiscoveryExecutionContract, provider.DiscoveryOutputContract
	}
	if input.Role == "sniper" && input.Slot == string(domain.SlotExecution) {
		executionContract, outputContract = provider.SniperExecutionContract, provider.SniperOutputContract
		_, status, loadErr := loadMission(input.Root, input.MissionID)
		if loadErr != nil {
			return domain.MissionInvocationRequest{}, time.Time{}, fmt.Errorf("load Sniper mission authorization: %w", loadErr)
		}
		approvalDigest = status.ApprovalGatePackageDigest
	}
	request, issued, err := application.NewInvocationRequest(ctx, application.InvocationRequestInput{
		Root: input.Root, BasePath: input.BasePath, MissionID: input.MissionID, Role: input.Role, Slot: input.Slot,
		RequestContext: input.RequestContext, Binding: binding, Weapon: weapon, Payload: payload,
		SourceDigest: sourceDigest, ApprovalGatePackageDigest: approvalDigest,
		ExecutionContract: executionContract, OutputContract: outputContract,
	})
	return request, issued, wrapMissionError(err)
}

func newInvocationID() (string, error) {
	id, err := application.NewInvocationID()
	return id, wrapMissionError(err)
}

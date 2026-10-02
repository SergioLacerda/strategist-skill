package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
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
	request, now, err := newMissionInvocationRequest(input, binding, weapon, payload, sourceDigest)
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

func newMissionInvocationRequest(input missionadapter.InvocationBuildInput, binding domain.RoleWeaponBinding, weapon domain.CompiledWeapon, payload []byte, sourceDigest string) (domain.MissionInvocationRequest, time.Time, error) {
	requestID, err := newInvocationID()
	if err != nil {
		return domain.MissionInvocationRequest{}, time.Time{}, err
	}
	now := time.Now().UTC()
	requestInput := map[string]any{}
	if input.Role == "ranger" {
		requestInput["execution_contract"] = provider.DiscoveryExecutionContract
		requestInput["output_contract"] = provider.DiscoveryOutputContract
	}
	if input.Role == "sniper" && input.Slot == string(domain.SlotExecution) {
		_, status, loadErr := loadMission(input.Root, input.MissionID)
		if loadErr != nil {
			return domain.MissionInvocationRequest{}, time.Time{}, fmt.Errorf("load Sniper mission authorization: %w", loadErr)
		}
		requestInput["execution_contract"] = provider.SniperExecutionContract
		requestInput["output_contract"] = provider.SniperOutputContract
		requestInput["refined_package"] = filepath.ToSlash(filepath.Join(input.BasePath, "refined", input.MissionID))
		requestInput["report_path"] = filepath.ToSlash(filepath.Join(input.BasePath, "archived", input.MissionID+"-report.md"))
		requestInput["approval_gate_package_digest"] = status.ApprovalGatePackageDigest
	}
	if strings.TrimSpace(input.RequestContext) != "" {
		requestInput["request_context"] = input.RequestContext
	}
	return domain.MissionInvocationRequest{
		Protocol: domain.MissionInvocationProtocolVersion, RequestID: requestID,
		MissionID: input.MissionID, Role: input.Role, Slot: input.Slot,
		Weapon:        domain.MissionWeaponIdentity{ID: weapon.ID, Version: weapon.Version, Digest: weapon.Digest},
		BindingDigest: binding.BindingDigest, SourceDigest: sourceDigest,
		ExecutionMode: binding.ExecutionMode, Entrypoint: binding.Entrypoint,
		Payload: string(payload), Input: requestInput, Nonce: newPromptNonce(),
	}, now, nil
}

func newInvocationID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("mission invocation: generate request id: %w", err)
	}
	return "inv_" + hex.EncodeToString(raw[:]), nil
}

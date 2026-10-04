package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/feats/initiative"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/roles/registry"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	telemetrysink "github.com/SergioLacerda/strategist-skill/internal/telemetry/sink"
	"github.com/SergioLacerda/strategist-skill/internal/tools/leveling"
	"github.com/spf13/cobra"
)

// missionComposition wires process-owned runtime dependencies into the
// mission Cobra adapter. Mission policy and lifecycle rules stay below this
// executable boundary.
func missionComposition() missionadapter.Composition {
	return missionadapter.Composition{
		Lifecycle:  missionLifecycleDependencies(),
		View:       missionViewDependencies(),
		Normalize:  missionNormalizeDependencies(),
		Usage:      missionReportUsageDependencies(),
		Invocation: missionInvocationDependencies(),
	}
}

func missionLifecycleDependencies() missionadapter.LifecycleDependencies {
	return missionadapter.LifecycleDependencies{
		RootFlag: cliutil.FlagRoot, RequireMissionID: requireMissionID,
		ResolveBasePath: cliutil.ResolveActiveBasePath, RequireNoExisting: requireNoExistingMission,
		Save: saveMission, Load: loadMission, InitiativeStart: startInitiativeConsultation,
		WriteResult: missionadapter.WriteResult, Lock: lockMission, ADRCanonicalPath: adrCanonicalPath,
		TelemetrySink: selectDiscoveryTelemetrySink,
		RecordRoute:   recordMissionRoute,
		ActivateCriticalHit: func(root, missionID string) (domain.MissionEngineStatus, error) {
			return missionruntime.ActivateCriticalHitRoute(root, missionID, loadMission, saveMission)
		},
	}
}

func missionViewDependencies() missionadapter.ViewDependencies {
	return missionadapter.ViewDependencies{
		RootFlag: cliutil.FlagRoot, Ledger: roleLevelLedger,
		RequireMissionID: requireMissionID, ResolveBasePath: cliutil.ResolveActiveBasePath,
		Load: loadMission, FilterLevels: filterMissionLevelRecords,
	}
}

func missionNormalizeDependencies() missionadapter.NormalizeDependencies {
	return missionadapter.NormalizeDependencies{
		RootFlag: cliutil.FlagRoot, RequireMissionID: validateMissionID,
		ResolvePaths: resolveNormalizePaths, RecordConfidence: recordNormalizeConfidence,
		RecordPublication: recordNormalizePublication, LoadMission: loadNormalizeMission,
		GateLabel: resolveNormalizeGateLabel,
	}
}

func missionReportUsageDependencies() missionadapter.ReportUsageDependencies {
	return missionadapter.ReportUsageDependencies{
		RootFlag: cliutil.FlagRoot, ResolveBasePath: cliutil.ResolveActiveBasePath,
		MissionKnown: missionIDKnown,
		SilenceRun: func(cmd *cobra.Command) {
			if run := cliutil.TelemetryRunFromCmd(cmd); run != nil {
				run.SetSilent()
			}
		},
		Validate: validateMissionReportUsageOptions,
	}
}

func missionInvocationDependencies() missionadapter.InvocationDependencies {
	return missionadapter.InvocationDependencies{
		RootFlag: cliutil.FlagRoot, RequireMissionID: requireMissionID,
		ResolveBasePath: resolveActiveBasePathForMission,
		LoadMission: func(root, missionID string) (domain.MissionEngineStatus, error) {
			_, status, err := loadMission(root, missionID)
			return status, err
		},
		Build: buildMissionInvocation, Complete: completeMissionInvocation,
		ListRequests: listMissionInvocationRequests,
		ExecuteHost:  executeMissionHost, WriteResult: missionadapter.WriteResult,
		ReadCompletion: missionadapter.ReadCompletion, TelemetrySink: selectDiscoveryTelemetrySink,
	}
}

// selectDiscoveryTelemetrySink applies the existing telemetry selection policy
// (slog or OTel, resilient per STRATEGIST_TELEMETRY_STRICT) to the environment
// read when the command runs. There is no external governance bridge in
// ordinary CLI composition, which selects no alternate sink.
func selectDiscoveryTelemetrySink() telemetry.EventSink {
	return telemetrysink.Select(telemetry.FromEnv(), nil)
}

func resolveActiveBasePathForMission(root string) (string, string, error) {
	strategistRoot, basePath, err := cliutil.ResolveActiveBasePath(root)
	if err != nil {
		return "", "", fmt.Errorf("resolve active base path: %w", err)
	}
	return strategistRoot, basePath, nil
}

func recordMissionRoute(ctx context.Context, root, missionID string, raw []byte) (bool, error) {
	appended, err := missionruntime.RecordRouteDecisionWithTelemetry(ctx, root, missionID, raw, selectDiscoveryTelemetrySink())
	if err != nil {
		return false, fmt.Errorf("record mission route: %w", err)
	}
	return appended, nil
}

func adrCanonicalPath(root string) (string, error) {
	cfg, err := cliutil.LoadActiveConfig(root)
	if err != nil {
		return "", fmt.Errorf("read adr.canonical_path: %w", err)
	}
	return cfg.ADR.CanonicalPath, nil
}

func startInitiativeConsultation(root, missionID string) error {
	registry, err := roles.LoadRoleRegistry(filepath.Join(root, "roles"))
	if err != nil {
		registry = domain.DefaultRoleRegistry()
	}
	lifecycle, err := missionruntime.NewDefaultRoleLifecycle(root, registry)
	if err != nil {
		return fmt.Errorf("create initiative lifecycle: %w", err)
	}
	levelingResolution, err := resolveMissionLeveling(root, missionID, "scout", missionID)
	if err != nil {
		return fmt.Errorf("resolve LEVELING before INITIATIVE: %w", err)
	}
	_, err = lifecycle.Enter(missionruntime.InitiativeRoleEntry{
		MissionID: missionID, Role: "scout", RunID: missionID, Leveling: &levelingResolution,
	})
	if err != nil {
		return fmt.Errorf("enter initiative role: %w", err)
	}
	return nil
}

func resolveMissionLeveling(root, missionID, role, runID string) (initiative.LevelingResolution, error) {
	path := filepath.Join(root, "memory", roleLevelLedger)
	record, found, err := leveling.LatestRunRecord(path, missionID, role, runID)
	if err != nil {
		return initiative.LevelingResolution{}, fmt.Errorf("read LEVELING ledger: %w", err)
	}
	if !found {
		record = leveling.Record{
			MissionID: missionID, Run: runID, Level: leveling.Level{Role: leveling.NormalizeRole(role)},
			Reason: "mission_start", Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		}
	}
	state := initiative.ObservationUnavailable
	if !record.Unknown() {
		state = initiative.ObservationKnown
	}
	return initiative.LevelingResolution{
		EventID: "leveling-" + missionID + "-" + runID + "-" + record.Timestamp,
		Role:    record.Role, State: state, Model: record.Model, Provider: record.Provider,
		Effort: initiative.EffortTier(record.Effort), Capability: record.Capability,
		LevelSource: record.Source, PolicyDigest: record.PolicyDigest,
	}, nil
}

func requireNoExistingMission(root, missionID string) error {
	if err := missionruntime.RequireNoExisting(root, missionID); err != nil {
		return fmt.Errorf("mission start: %w", err)
	}
	return nil
}

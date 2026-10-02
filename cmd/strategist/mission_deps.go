package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/initiative"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/spf13/cobra"
)

func missionLifecycleDependencies() missionadapter.LifecycleDependencies {
	return missionadapter.LifecycleDependencies{
		RootFlag: cliutil.FlagRoot, RequireMissionID: requireMissionID,
		ResolveBasePath: cliutil.ResolveActiveBasePath, RequireNoExisting: requireNoExistingMission,
		Save: saveMission, Load: loadMission, InitiativeStart: startInitiativeConsultation,
		WriteResult: writeMissionResult, Lock: lockMission, ADRCanonicalPath: adrCanonicalPath,
	}
}

func adrCanonicalPath(root string) (string, error) {
	cfg, err := cliutil.LoadActiveConfig(root)
	if err != nil {
		return "", fmt.Errorf("read adr.canonical_path: %w", err)
	}
	return cfg.ADR.CanonicalPath, nil
}

func startInitiativeConsultation(root, missionID string) error {
	registry, err := domain.LoadRoleRegistry(filepath.Join(root, "roles"))
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
		MissionID: missionID,
		Role:      "scout",
		RunID:     missionID,
		Leveling:  &levelingResolution,
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
		// No prior ledger entry, and this call site has no host or provider
		// data to resolve from (mission start, before the role's own on_start
		// hook ever runs) — leveling.ResolveLevel(Policy{}, "", role, ...,
		// Host{}) with everything empty can only ever return an Unknown()
		// level, so it, its append, and its role_level_resolved log line are
		// skipped rather than recording a tuple that says nothing (F-L4,
		// ADR-0057/design.md task 4.4). Any distribution computed over the
		// ledger is unaffected: a row with empty model/effort/level_source
		// carried no information to begin with.
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
		Role:    record.Role, State: state, Model: record.Model,
		Provider: record.Provider, Effort: initiative.EffortTier(record.Effort),
		Capability: record.Capability, LevelSource: record.Source,
		PolicyDigest: record.PolicyDigest,
	}, nil
}

func missionViewDependencies() missionadapter.ViewDependencies {
	return missionadapter.ViewDependencies{RootFlag: cliutil.FlagRoot, Ledger: roleLevelLedger, RequireMissionID: requireMissionID, ResolveBasePath: cliutil.ResolveActiveBasePath, Load: loadMission, FilterLevels: filterMissionLevelRecords}
}

func missionNormalizeDependencies() missionadapter.NormalizeDependencies {
	return missionadapter.NormalizeDependencies{
		RootFlag: cliutil.FlagRoot, RequireMissionID: validateMissionID,
		ResolvePaths: resolveNormalizePaths, RecordConfidence: recordNormalizeConfidence, RecordPublication: recordNormalizePublication, LoadMission: loadNormalizeMission,
		GateLabel: resolveNormalizeGateLabel,
	}
}

func missionReportUsageDependencies() missionadapter.ReportUsageDependencies {
	return missionadapter.ReportUsageDependencies{RootFlag: cliutil.FlagRoot, ResolveBasePath: cliutil.ResolveActiveBasePath, MissionKnown: missionIDKnown, SilenceRun: func(cmd *cobra.Command) {
		if run := cliutil.TelemetryRunFromCmd(cmd); run != nil {
			run.SetSilent()
		}
	}, Validate: validateMissionReportUsageOptions}
}

func requireNoExistingMission(root, missionID string) error {
	if _, err := os.Stat(missionPath(root, missionID)); err == nil {
		return fmt.Errorf("mission start: mission %q already exists", missionID)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("mission start: inspect existing state: %w", err)
	}
	return nil
}

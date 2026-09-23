package main

import (
	"fmt"
	"os"
	"path/filepath"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/initiative"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/spf13/cobra"
)

func missionLifecycleDependencies() missionadapter.LifecycleDependencies {
	return missionadapter.LifecycleDependencies{
		RootFlag: flagRoot, RequireMissionID: requireMissionID,
		ResolveBasePath: cliutil.ResolveActiveBasePath, RequireNoExisting: requireNoExistingMission,
		Save: saveMission, Load: loadMission, InitiativeStart: startInitiativeConsultation,
		WriteResult: writeMissionResult,
	}
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
	_, err = lifecycle.Enter(missionruntime.InitiativeRoleEntry{
		MissionID: missionID,
		Role:      "scout",
		RunID:     missionID,
		Observed:  initiative.Observation{State: initiative.ObservationUnavailable},
	})
	if err != nil {
		return fmt.Errorf("enter initiative role: %w", err)
	}
	return nil
}

func missionViewDependencies() missionadapter.ViewDependencies {
	return missionadapter.ViewDependencies{RootFlag: flagRoot, Ledger: roleLevelLedger, RequireMissionID: requireMissionID, ResolveBasePath: cliutil.ResolveActiveBasePath, Load: loadMission, FilterLevels: filterMissionLevelRecords}
}

func missionNormalizeDependencies() missionadapter.NormalizeDependencies {
	return missionadapter.NormalizeDependencies{RootFlag: flagRoot, RequireMissionID: validateMissionID, ResolvePaths: resolveNormalizePaths}
}

func missionReportUsageDependencies() missionadapter.ReportUsageDependencies {
	return missionadapter.ReportUsageDependencies{RootFlag: flagRoot, ResolveBasePath: cliutil.ResolveActiveBasePath, MissionKnown: missionIDKnown, SilenceRun: func(cmd *cobra.Command) {
		if run := telemetryRunFromCmd(cmd); run != nil {
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

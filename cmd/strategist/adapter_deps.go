package main

import (
	"fmt"
	"os"

	installadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/install"
	metricsadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/metrics"
	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	internalinstall "github.com/SergioLacerda/strategist-skill/internal/install"
	"github.com/spf13/cobra"
)

func missionLifecycleDependencies() missionadapter.LifecycleDependencies {
	return missionadapter.LifecycleDependencies{RootFlag: flagRoot, RequireMissionID: requireMissionID, ResolveBasePath: cliutil.ResolveActiveBasePath, RequireNoExisting: requireNoExistingMission, Save: saveMission, Load: loadMission, WriteResult: writeMissionResult}
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

const defaultLedgerMaxRecords = 2000

func metricsDependencies() metricsadapter.Dependencies {
	return metricsadapter.Dependencies{
		RootFlag:    flagRoot,
		ResolveRoot: resolveMetricsRoot,
		SilenceRun: func(cmd *cobra.Command) {
			if run := telemetryRunFromCmd(cmd); run != nil {
				run.SetSilent()
			}
		},
	}
}

func installDependencies() installadapter.Dependencies {
	return installadapter.Dependencies{
		ResolveTarget:  resolveRuntimeInstallTarget,
		UserHomeDir:    os.UserHomeDir,
		ServiceFactory: newInstallService,
	}
}

func resolveRuntimeInstallTarget(explicit string, global bool) (string, error) {
	target, err := installadapter.ResolveTarget(explicit, global, findStrategistRoot)
	if err != nil {
		return "", fmt.Errorf("resolve install target: %w", err)
	}
	return target, nil
}

func newInstallService(shimHome string) installadapter.Installer {
	return internalinstall.Service{
		Extractor:          embedpkg.Extractor{},
		Lister:             embedpkg.Extractor{},
		Compiler:           compile.Compiler{},
		ShimHomeDir:        shimHome,
		AwarenessRefresher: refreshAgentAwarenessFromEmbed,
		Version:            Version,
	}
}

func refreshAgentAwarenessFromEmbed(strategistRoot, projectRoot, version string) bool {
	tplBytes, err := embedpkg.Extractor{}.ReadFile("templates/agent-protocol.md")
	if err != nil {
		tplBytes = nil
	}
	return compile.RefreshAgentAwareness(strategistRoot, projectRoot, version, tplBytes)
}

var _ domain.FileExtractor = embedpkg.Extractor{}

func resolveMetricsRoot(cmd *cobra.Command, action, explicitRoot string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("metrics %s: get cwd: %w", action, err)
	}
	rootInput := explicitRoot
	if rootInput == "" {
		rootInput = stringFlag(cmd, flagRoot, "")
	}
	root, _, err := resolveStrategistRoot(rootInput, cwd)
	if err != nil {
		return "", fmt.Errorf("metrics %s: %w", action, err)
	}
	return root, nil
}

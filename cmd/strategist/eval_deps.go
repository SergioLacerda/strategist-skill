package main

import (
	"fmt"
	"os"

	evaladapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/eval"
	"github.com/spf13/cobra"
)

// resolveEvalActionRoot resolves both the .strategist root and its parent
// project root for an eval subcommand. Unlike resolveMetricsRoot, eval
// subcommands may write outside .strategist/ (e.g. harvest writes to
// tests/evals/regression/ under the project root), so both are returned.
func resolveEvalActionRoot(cmd *cobra.Command, action, explicitRoot string) (strategistRoot, projectRoot string, err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("eval %s: get cwd: %w", action, err)
	}
	strategistRoot, projectRoot, err = resolveStrategistRoot(stringFlag(cmd, flagRoot, explicitRoot), cwd)
	if err != nil {
		return "", "", fmt.Errorf("eval %s: %w", action, err)
	}
	return strategistRoot, projectRoot, nil
}

func evalDependencies() evaladapter.Dependencies {
	return evaladapter.Dependencies{
		RootFlag:    flagRoot,
		ResolveRoot: resolveEvalActionRoot,
		SilenceRun: func(cmd *cobra.Command) {
			if run := telemetryRunFromCmd(cmd); run != nil {
				run.SetSilent()
			}
		},
	}
}

func evalHarvestDependencies() evaladapter.HarvestDependencies {
	return evaladapter.HarvestDependencies{
		RootFlag:        flagRoot,
		ResolveRoot:     resolveEvalActionRoot,
		ResolveBasePath: resolveDojoRoots,
		Select:          evaladapter.SelectHarvestMissionIDs,
		PrintWarnings:   evaladapter.PrintHarvestWarnings,
		ParseInclude:    evaladapter.ParseHarvestInclude,
		Harvest:         evaladapter.HarvestMissions,
		SilenceRun: func(cmd *cobra.Command) {
			if run := telemetryRunFromCmd(cmd); run != nil {
				run.SetSilent()
			}
		},
	}
}

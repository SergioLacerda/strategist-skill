package main

import (
	"fmt"
	"os"

	metricsadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/metrics"
	"github.com/spf13/cobra"
)

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

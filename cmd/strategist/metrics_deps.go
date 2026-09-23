package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	metricsadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/metrics"
	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/spf13/cobra"
)

func metricsDependencies() metricsadapter.Dependencies {
	return metricsadapter.Dependencies{
		RootFlag:        cliutil.FlagRoot,
		ResolveRoot:     resolveMetricsRoot,
		ResolveBasePath: resolveMetricsBasePath,
		SilenceRun: func(cmd *cobra.Command) {
			if run := cliutil.TelemetryRunFromCmd(cmd); run != nil {
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
		rootInput = cliutil.StringFlag(cmd, cliutil.FlagRoot, "")
	}
	root, _, err := resolveStrategistRoot(rootInput, cwd)
	if err != nil {
		return "", fmt.Errorf("metrics %s: %w", action, err)
	}
	return root, nil
}

// resolveMetricsBasePath returns the workspace base_path for a resolved
// .strategist/ root. A runtime without active.yaml has no workspace artifact
// tree to protect, so it yields "" rather than an error; any other failure is
// reported so the claim-location guard never silently turns off.
func resolveMetricsBasePath(strategistRoot string) (string, error) {
	_, basePath, err := cliutil.ResolveActiveBasePath(strategistRoot)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("resolve active base path: %w", err)
	}
	return basePath, nil
}

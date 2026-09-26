package main

import (
	"fmt"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"path/filepath"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
)

func resolveNormalizePaths(opts missionadapter.NormalizeOptions) (string, string, string, error) {
	strategistRoot, basePath, err := cliutil.ResolveActiveBasePath(opts.Root)
	if err != nil {
		return "", "", "", fmt.Errorf("resolve active base path: %w", err)
	}
	projectRoot := filepath.Dir(strategistRoot)
	return basePath, resolvePath(opts.RuntimeRoot, filepath.Join(strategistRoot, "openspec"), projectRoot),
		resolvePath(opts.Pending, filepath.Join(basePath, "pending", opts.MissionID+"-analysis.md"), projectRoot), nil
}

func resolvePath(value, fallback, projectRoot string) string {
	if value == "" {
		return fallback
	}
	if filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(projectRoot, value)
}

// resolveNormalizeGateLabel reads the mission's gate outcome label, which an
// amendment consults: a rejected analysis is not amended.
func resolveNormalizeGateLabel(opts missionadapter.NormalizeOptions) (string, error) {
	strategistRoot, _, err := cliutil.ResolveActiveBasePath(opts.Root)
	if err != nil {
		return "", fmt.Errorf("resolve active base path: %w", err)
	}
	label, err := telemetry.GateOutcomeFor(strategistRoot, opts.MissionID)
	if err != nil {
		return "", fmt.Errorf("gate outcome: %w", err)
	}
	return label, nil
}

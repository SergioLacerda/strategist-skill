// Package cliutil holds small CLI helpers shared across cmd/strategist's own
// package-main command files and any importable command packages (e.g.
// internal/treasurecli) that need the same root-resolution and flag-reading
// behavior a package-main file cannot export to them directly.
package cliutil

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Output format and flag-name constants shared by cmd/strategist and any
// importable command package (e.g. internal/treasurecli).
const (
	OutputFormatTable = "table"
	OutputFormatJSON  = "json"

	FlagFormat            = "format"
	FlagRoot              = "root"
	FlagIndex             = "index"
	FlagIncludeHistorical = "include-historical"
)

const (
	strategistDirName      = ".strategist"
	rootDiscoveryMaxLevels = 5
)

// FindStrategistRoot walks up from startDir looking for a .strategist/ directory.
// Returns (strategistDir, projectRoot, nil) on success.
// strategistDir is the full path to .strategist/.
// projectRoot is its parent (the project root).
// Returns an error if no .strategist/ is found within rootDiscoveryMaxLevels levels.
func FindStrategistRoot(startDir string) (strategistDir, projectRoot string, err error) {
	dir := startDir
	for i := 0; i < rootDiscoveryMaxLevels; i++ {
		candidate := filepath.Join(dir, strategistDirName)
		if stat, statErr := os.Stat(candidate); statErr == nil && stat.IsDir() {
			return candidate, dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		dir = parent
	}
	return "", "", fmt.Errorf("%s not found within %d levels of %s — run: strategist install", strategistDirName, rootDiscoveryMaxLevels, startDir)
}

// ResolveStrategistRoot returns the runtime root to use for a command.
// If explicit is non-empty, it is used directly (projectRoot = its parent).
// Otherwise, FindStrategistRoot is called from cwd.
func ResolveStrategistRoot(explicit, cwd string) (strategistDir, projectRoot string, err error) {
	if explicit != "" {
		abs, absErr := filepath.Abs(explicit)
		if absErr != nil {
			return "", "", fmt.Errorf("resolve root: %w", absErr)
		}
		return abs, filepath.Dir(abs), nil
	}
	return FindStrategistRoot(cwd)
}

// StringFlag reads a string flag from cmd, checking inherited flags as a fallback,
// and returns fallback if neither is set.
func StringFlag(cmd *cobra.Command, name, fallback string) string {
	if cmd == nil {
		return fallback
	}
	value, err := cmd.Flags().GetString(name)
	if err == nil {
		return value
	}
	value, err = cmd.InheritedFlags().GetString(name)
	if err == nil {
		return value
	}
	return fallback
}

// BoolFlag reads a bool flag from cmd, checking inherited flags as a fallback,
// and returns fallback if neither is set.
func BoolFlag(cmd *cobra.Command, name string, fallback bool) bool {
	if cmd == nil {
		return fallback
	}
	value, err := cmd.Flags().GetBool(name)
	if err == nil {
		return value
	}
	value, err = cmd.InheritedFlags().GetBool(name)
	if err == nil {
		return value
	}
	return fallback
}

// ResolveActiveBasePath reads active.yaml from strategistRoot (defaulting to
// ".strategist" when empty) and resolves its base_path to an absolute-or-
// root-relative path. Returns the resolved strategistRoot and basePath.
func ResolveActiveBasePath(root string) (strategistRoot, basePath string, err error) {
	strategistRoot = root
	if strategistRoot == "" {
		strategistRoot = ".strategist"
	}

	activeYamlPath, err := runtimefs.SafeJoin(strategistRoot, "active.yaml")
	if err != nil {
		return "", "", fmt.Errorf("resolve active.yaml path: %w", err)
	}
	raw, err := os.ReadFile(activeYamlPath) //nolint:gosec // G304: path validated by runtimefs.SafeJoin, confined to strategistRoot
	if err != nil {
		return "", "", fmt.Errorf("read active.yaml: %w", err)
	}
	var cfg domain.ActiveConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return "", "", fmt.Errorf("parse active.yaml: %w", err)
	}
	if cfg.BasePath == "" {
		return "", "", fmt.Errorf("active.yaml: base_path is empty")
	}
	basePath = cfg.BasePath
	if !filepath.IsAbs(basePath) {
		basePath = filepath.Join(filepath.Dir(strategistRoot), basePath)
	}
	return strategistRoot, basePath, nil
}

// TelemetryRunFromCmd extracts the MissionRun from the command context, if any.
func TelemetryRunFromCmd(cmd *cobra.Command) *telemetry.MissionRun {
	if cmd == nil {
		return nil
	}
	ctx := cmd.Context()
	if ctx == nil {
		return nil
	}
	return telemetry.MissionRunFromContext(ctx)
}

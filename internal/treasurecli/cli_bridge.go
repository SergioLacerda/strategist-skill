package treasurecli

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// Thin wrappers over internal/cliutil, mirroring cmd/strategist's own
// cli_helpers.go — both packages share the same internal/cliutil source of
// truth for root resolution and flag reading, without either importing the
// other (cmd/strategist is package main and imports this package; the
// reverse would be an import cycle).

const (
	outputFormatTable = cliutil.OutputFormatTable
	outputFormatJSON  = cliutil.OutputFormatJSON

	flagFormat            = cliutil.FlagFormat
	flagRoot              = cliutil.FlagRoot
	flagIndex             = cliutil.FlagIndex
	flagIncludeHistorical = cliutil.FlagIncludeHistorical
)

func resolveStrategistRoot(explicit, cwd string) (strategistDir, projectRoot string, err error) {
	return cliutil.ResolveStrategistRoot(explicit, cwd) //nolint:wrapcheck // pure delegation; cliutil's error text is this function's own contract, preserved verbatim on purpose
}

// resolveDojoRoots delegates to internal/cliutil, preserving the exact "dojo: "
// error prefix the pre-move code produced (the underlying logic — read
// active.yaml, resolve base_path — is generic, not dojo-specific; only the
// original name and its error prefix stuck, and callers here already wrap
// this error further, e.g. "treasure-chest index: %w").
func resolveDojoRoots(root string) (strategistRoot, basePath string, err error) {
	strategistRoot, basePath, err = cliutil.ResolveActiveBasePath(root)
	if err != nil {
		return "", "", fmt.Errorf("dojo: %w", err)
	}
	return strategistRoot, basePath, nil
}

func stringFlag(cmd *cobra.Command, name, fallback string) string {
	return cliutil.StringFlag(cmd, name, fallback)
}

func boolFlag(cmd *cobra.Command, name string, fallback bool) bool {
	return cliutil.BoolFlag(cmd, name, fallback)
}

func telemetryRunFromCmd(cmd *cobra.Command) *telemetry.MissionRun {
	return cliutil.TelemetryRunFromCmd(cmd)
}

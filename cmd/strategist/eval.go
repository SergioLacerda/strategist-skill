package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// evalCmd is the parent for Strategist's internal/eval harness utilities.
// Subcommands live in their own file, one per action (harvest here in
// eval_harvest.go), mirroring metrics.go/metrics_scout.go's grouping.
var evalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Strategist eval harness utilities",
}

// resolveEvalActionRoot resolves both the .strategist root and its parent
// project root for an eval subcommand. Unlike resolveMetricsActionRoot,
// eval subcommands may write outside .strategist/ (e.g. harvest writes to
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

func init() {
	rootCmd.AddCommand(evalCmd)
}

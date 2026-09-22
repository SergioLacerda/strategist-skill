package main

import (
	"fmt"
	"os"

	evaladapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/eval"
	"github.com/spf13/cobra"
)

// evalCmd is the parent for Strategist's internal/eval harness utilities.
// Subcommands live in their own file, one per action (harvest here in
// eval_harvest.go), mirroring metrics.go/metrics_scout.go's grouping.
var evalCmd = evaladapter.NewParent()

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

// registerEval attaches eval at the root composition boundary. Evaluation and
// fixture-selection behavior remain owned by internal/eval.
func registerEval(root *cobra.Command) {
	evalCmd.AddCommand(evalRunCmd, evalHarvestCmd)
	root.AddCommand(evalCmd)
}

func newEvalCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "eval", Short: "Strategist eval harness utilities"}
	cmd.AddCommand(newEvalRunCommand(), newEvalHarvestCommand())
	return cmd
}

func evalAdapterDependencies() evaladapter.Dependencies {
	return evaladapter.Dependencies{
		RootFlag: flagRoot, ResolveRoot: resolveEvalActionRoot,
		SilenceRun: func(cmd *cobra.Command) {
			if run := telemetryRunFromCmd(cmd); run != nil {
				run.SetSilent()
			}
		},
	}
}

package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// evalRunCmd executes the internal/eval scenario battery via go test.
// Design: .analysis/refined/20260804-eval-cli-subcommand/design.md.
// Decisions (DEC-1..3): .analysis/archived/20260804-eval-cli-subcommand-adr.md.
var evalRunCmd = &cobra.Command{
	Use:   "run [pattern]",
	Short: "Run the internal/eval scenario battery via go test",
	Long: `Run Strategist's tagged eval scenario battery: go test -tags=eval <pattern>,
defaulting to ./tests/evals/... when no pattern is given. Equivalent to "make eval"
when run with no arguments. Shells out to the go toolchain — requires "go" on PATH.`,
}

type evalRunOptions struct {
	Root string
	Race bool
}

func runEvalRun(cmd *cobra.Command, args []string, opts evalRunOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	opts.Race = boolFlag(cmd, "race", opts.Race)
	opts.Root = stringFlag(cmd, flagRoot, opts.Root)

	_, projectRoot, err := resolveEvalActionRoot(cmd, "run", opts.Root)
	if err != nil {
		return err
	}

	goArgs := buildEvalRunGoTestArgs(resolveEvalRunPattern(args), opts.Race)

	goTestCmd := exec.Command("go", goArgs...) //nolint:gosec // G204: args are a fixed literal ("test", "-race", "-tags=eval") plus a caller-supplied Go package pattern, not arbitrary shell input
	goTestCmd.Dir = projectRoot
	goTestCmd.Stdout = os.Stdout
	goTestCmd.Stderr = os.Stderr
	if err := goTestCmd.Run(); err != nil {
		return fmt.Errorf("go test: %w", err)
	}
	return nil
}

// resolveEvalRunPattern returns the Go package pattern to test: the first
// non-empty positional argument, or "./tests/evals/..." when none is given.
func resolveEvalRunPattern(args []string) string {
	if len(args) > 0 && args[0] != "" {
		return args[0]
	}
	return "./tests/evals/..."
}

// buildEvalRunGoTestArgs builds the "go test" argument list: -race is
// included only when race is true (default), then -tags=eval and pattern
// always follow, matching "make eval"'s own invocation shape.
func buildEvalRunGoTestArgs(pattern string, race bool) []string {
	args := []string{"test"}
	if race {
		args = append(args, "-race")
	}
	return append(args, "-tags=eval", pattern)
}

func init() {
	opts := evalRunOptions{}
	evalRunCmd.Flags().BoolVar(&opts.Race, "race", true, "pass -race to go test")
	evalRunCmd.Flags().StringVar(&opts.Root, flagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	evalRunCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runEvalRun(cmd, args, opts)
	}
	evalCmd.AddCommand(evalRunCmd)
}

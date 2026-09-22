package main

import (
	"fmt"

	evaladapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/eval"
	"github.com/spf13/cobra"
)

// evalRunCmd executes the internal/eval scenario battery via go test.
// Design: .analysis/refined/20260804-eval-cli-subcommand/design.md.
// Decisions (DEC-1..3): .analysis/archived/20260804-eval-cli-subcommand-adr.md.
var evalRunCmd = newEvalRunCommand()

func newEvalRunCommand() *cobra.Command {
	return evaladapter.NewRun(evalAdapterDependencies())
}

type evalRunOptions struct {
	Root string
	Race bool
}

func runEvalRun(cmd *cobra.Command, args []string, opts evalRunOptions) error {
	if err := evaladapter.Run(cmd, args, evalAdapterDependencies(), opts.Root, opts.Race); err != nil {
		return fmt.Errorf("run eval run: %w", err)
	}
	return nil
}

// resolveEvalRunPattern returns the Go package pattern to test: the first
// non-empty positional argument, or "./tests/evals/..." when none is given.
func resolveEvalRunPattern(args []string) string {
	return evaladapter.ResolvePattern(args)
}

// buildEvalRunGoTestArgs builds the "go test" argument list: -race is
// included only when race is true (default), then -tags=eval and pattern
// always follow, matching "make eval"'s own invocation shape.
func buildEvalRunGoTestArgs(pattern string, race bool) []string {
	return evaladapter.BuildGoTestArgs(pattern, race)
}

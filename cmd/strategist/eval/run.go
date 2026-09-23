package eval

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// Dependencies defines injected dependencies for the eval run command.
type Dependencies struct {
	RootFlag    string
	ResolveRoot func(*cobra.Command, string, string) (string, string, error)
	SilenceRun  func(*cobra.Command)
}

// NewRun creates a new Cobra command to run eval scenario tests via go test.
func NewRun(deps Dependencies) *cobra.Command {
	var root string
	var race bool
	cmd := &cobra.Command{Use: "run [pattern]", Short: "Run the internal/eval scenario battery via go test"}
	cmd.Flags().BoolVar(&race, "race", true, "pass -race to go test")
	cmd.Flags().StringVar(&root, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	cmd.RunE = func(cmd *cobra.Command, args []string) error { return Run(cmd, args, deps, root, race) }
	return cmd
}

// Run executes the eval run command logic.
func Run(cmd *cobra.Command, args []string, deps Dependencies, root string, race bool) error {
	if deps.SilenceRun != nil {
		deps.SilenceRun(cmd)
	}
	_, projectRoot, err := deps.ResolveRoot(cmd, "run", root)
	if err != nil {
		return err
	}
	goArgs := BuildGoTestArgs(ResolvePattern(args), race)
	// #nosec G204 -- controlled arguments for internal go test runner
	goTestCmd := exec.Command("go", goArgs...)
	goTestCmd.Dir = projectRoot
	goTestCmd.Stdout = os.Stdout
	goTestCmd.Stderr = os.Stderr
	if err := goTestCmd.Run(); err != nil {
		return fmt.Errorf("go test: %w", err)
	}
	return nil
}

// ResolvePattern extracts the target pattern from args or returns default.
func ResolvePattern(args []string) string {
	if len(args) > 0 && args[0] != "" {
		return args[0]
	}
	return "./tests/evals/..."
}

// BuildGoTestArgs constructs arguments for go test invocation.
func BuildGoTestArgs(pattern string, race bool) []string {
	args := []string{"test"}
	if race {
		args = append(args, "-race")
	}
	return append(args, "-tags=eval", pattern)
}

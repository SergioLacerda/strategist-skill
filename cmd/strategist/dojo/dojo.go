// Package dojo contains the Cobra adapter for the Strategist dojo health-check
// scenarios. Scenario loading, running and persistence stay in internal/dojo.
package dojo

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	internaldojo "github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
)

// Dependencies supplies host-specific concerns at the CLI composition edge.
type Dependencies struct {
	// ResolveRoots maps an optional --root value to the .strategist/ root and
	// the workspace base_path that holds the dojo scenarios.
	ResolveRoots func(root string) (strategistRoot, basePath string, err error)
}

const rootFlagUsage = "path to .strategist/ root (default: .strategist)"

// New creates the dojo command tree with no package-level flag state.
func New(deps Dependencies) *cobra.Command {
	parent := &cobra.Command{
		Use:   "dojo",
		Short: "Health-check scenarios for the Strategist skill",
		Long: `Dojo validates that Strategist is correctly installed, configured, and operating.

Run a scenario check:
  strategist dojo check <scenario>

List available scenarios:
  strategist dojo list`,
	}
	parent.AddCommand(newCheck(deps), newList(deps))
	return parent
}

// Register attaches the dojo command at the supplied root.
func Register(root *cobra.Command, deps Dependencies) {
	root.AddCommand(New(deps))
}

func newCheck(deps Dependencies) *cobra.Command {
	var root string
	var filesOnly bool
	cmd := &cobra.Command{
		Use:   "check <scenario>",
		Short: "Run offline checks for a dojo scenario",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheck(cmd, deps, root, filesOnly, args[0])
		},
	}
	cmd.Flags().StringVar(&root, "root", "", rootFlagUsage)
	cmd.Flags().BoolVar(&filesOnly, "files-only", false,
		"skip checks that require an emit.log run (emit_log, timing, pipeline)")
	return cmd
}

func newList(deps Dependencies) *cobra.Command {
	var root string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available dojo scenarios",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd, deps, root)
		},
	}
	cmd.Flags().StringVar(&root, "root", "", rootFlagUsage)
	return cmd
}

func resolveRoots(deps Dependencies, root string) (strategistRoot, basePath string, err error) {
	if deps.ResolveRoots == nil {
		return "", "", fmt.Errorf("dojo: root resolver is not configured")
	}
	return deps.ResolveRoots(root)
}

func runCheck(cmd *cobra.Command, deps Dependencies, root string, filesOnly bool, scenario string) error {
	strategistRoot, basePath, err := resolveRoots(deps, root)
	if err != nil {
		return err
	}

	scenarioDir := filepath.Join(basePath, "dojo", scenario)
	criteria, err := internaldojo.LoadCriteria(scenarioDir)
	if err != nil {
		return fmt.Errorf("dojo check: %w", err)
	}

	emitLogPath := filepath.Join(basePath, "dojo", ".last-run", scenario, "emit.log")
	startedAt := time.Now()
	result := internaldojo.Run(criteria, basePath, strategistRoot, emitLogPath, filesOnly)
	finishedAt := time.Now()

	if err := printResult(cmd.OutOrStdout(), result); err != nil {
		return err
	}
	persistResult(cmd.ErrOrStderr(), basePath, result, startedAt, finishedAt)

	if !result.Passed() {
		return fmt.Errorf("dojo check: scenario %q failed (%d checks failed)", scenario, result.FailCount())
	}
	return nil
}

// persistResult writes learning artifacts (result.json, .history.jsonl, and — for
// failed runs — lesson.md) under the dojo storage domain. Persistence failures are
// reported to errOut but never change the check's pass/fail exit status: dojo is a
// health check first, a learning tool second.
func persistResult(errOut io.Writer, basePath string, result domain.DojoCheckResult, startedAt, finishedAt time.Time) {
	if err := internaldojo.PersistResult(basePath, result, startedAt, finishedAt); err != nil {
		fmt.Fprintf(errOut, "dojo: warning: failed to persist result: %v\n", err) //nolint:errcheck // best-effort warning: a failed stderr write must not change the check result
	}
	if err := internaldojo.WriteLesson(basePath, result); err != nil {
		fmt.Fprintf(errOut, "dojo: warning: failed to write lesson: %v\n", err) //nolint:errcheck // best-effort warning: a failed stderr write must not change the check result
	}
}

func runList(cmd *cobra.Command, deps Dependencies, root string) error {
	_, basePath, err := resolveRoots(deps, root)
	if err != nil {
		return err
	}

	dojoDir := filepath.Join(basePath, "dojo")
	entries, err := os.ReadDir(dojoDir)
	if err != nil {
		return fmt.Errorf("dojo list: read %s: %w", dojoDir, err)
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
	if err := writeListRows(w, dojoDir, entries); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("dojo list: flush: %w", err)
	}
	return nil
}

func writeListRows(w *tabwriter.Writer, dojoDir string, entries []os.DirEntry) error {
	for _, e := range entries {
		if !isScenarioEntry(dojoDir, e) {
			continue
		}
		if _, err := fmt.Fprintf(w, "%s\t%s\n", e.Name(), internaldojo.ScenarioDescription(dojoDir, e.Name())); err != nil {
			return fmt.Errorf("dojo list: write: %w", err)
		}
	}
	return nil
}

func isScenarioEntry(dojoDir string, e os.DirEntry) bool {
	if !e.IsDir() || e.Name() == ".last-run" {
		return false
	}
	return internaldojo.ScenarioHasCriteria(dojoDir, e.Name())
}

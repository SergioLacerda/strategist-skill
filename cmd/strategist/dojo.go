package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	dojoRoot      string
	dojoFilesOnly bool
)

var dojoCmd = &cobra.Command{
	Use:   "dojo",
	Short: "Health-check scenarios for the Strategist skill",
	Long: `Dojo validates that Strategist is correctly installed, configured, and operating.

Run a scenario check:
  strategist dojo check <scenario>

List available scenarios:
  strategist dojo list`,
}

var dojoCheckCmd = &cobra.Command{
	Use:   "check <scenario>",
	Short: "Run offline checks for a dojo scenario",
	Args:  cobra.ExactArgs(1),
	RunE:  runDojoCheck,
}

var dojoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available dojo scenarios",
	RunE:  runDojoList,
}

func runDojoCheck(_ *cobra.Command, args []string) error {
	scenario := args[0]

	strategistRoot, basePath, err := resolveDojoRoots(dojoRoot)
	if err != nil {
		return err
	}

	scenarioDir := filepath.Join(basePath, "dojo", scenario)
	criteria, err := dojo.LoadCriteria(scenarioDir)
	if err != nil {
		return fmt.Errorf("dojo check: %w", err)
	}

	emitLogPath := filepath.Join(basePath, "dojo", ".last-run", scenario, "emit.log")
	startedAt := time.Now()
	result := dojo.Run(criteria, basePath, strategistRoot, emitLogPath, dojoFilesOnly)
	finishedAt := time.Now()

	if err := printDojoResult(result); err != nil {
		return err
	}
	persistDojoResult(basePath, result, startedAt, finishedAt)

	if !result.Passed() {
		return fmt.Errorf("dojo check: scenario %q failed (%d checks failed)", scenario, result.FailCount())
	}
	return nil
}

// persistDojoResult writes learning artifacts (result.json, .history.jsonl, and — for
// failed runs — lesson.md) under the dojo storage domain. Persistence failures are
// reported to stderr but never change the check's pass/fail exit status: dojo is a
// health check first, a learning tool second.
func persistDojoResult(basePath string, result domain.DojoCheckResult, startedAt, finishedAt time.Time) {
	if err := dojo.PersistResult(basePath, result, startedAt, finishedAt); err != nil {
		fmt.Fprintf(os.Stderr, "dojo: warning: failed to persist result: %v\n", err)
	}
	if err := dojo.WriteLesson(basePath, result); err != nil {
		fmt.Fprintf(os.Stderr, "dojo: warning: failed to write lesson: %v\n", err)
	}
}

func runDojoList(_ *cobra.Command, _ []string) error {
	_, basePath, err := resolveDojoRoots(dojoRoot)
	if err != nil {
		return err
	}

	dojoDir := filepath.Join(basePath, "dojo")
	entries, err := os.ReadDir(dojoDir)
	if err != nil {
		return fmt.Errorf("dojo list: read %s: %w", dojoDir, err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if err := writeDojoListRows(w, dojoDir, entries); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("dojo list: flush: %w", err)
	}
	return nil
}

func writeDojoListRows(w *tabwriter.Writer, dojoDir string, entries []os.DirEntry) error {
	for _, e := range entries {
		if !isDojoScenarioEntry(dojoDir, e) {
			continue
		}
		if _, err := fmt.Fprintf(w, "%s\t%s\n", e.Name(), dojoDescription(dojoDir, e.Name())); err != nil {
			return fmt.Errorf("dojo list: write: %w", err)
		}
	}
	return nil
}

func isDojoScenarioEntry(dojoDir string, e os.DirEntry) bool {
	if !e.IsDir() || e.Name() == ".last-run" {
		return false
	}
	return dojo.ScenarioHasCriteria(dojoDir, e.Name())
}

func dojoDescription(dojoDir, scenario string) string {
	criteriaPath := filepath.Join(dojoDir, scenario, "criteria.yaml")
	raw, err := os.ReadFile(criteriaPath) //nolint:gosec // G304: dojo list reads criteria.yaml under a discovered scenario directory
	if err != nil {
		return ""
	}
	var c domain.DojoCriteria
	if yaml.Unmarshal(raw, &c) != nil {
		return ""
	}
	return c.Description
}

// resolveDojoRoots delegates to internal/cliutil (shared with internal/treasurecli
// — see its own cli_bridge.go), re-adding the "dojo: " error prefix this
// function's own callers have always returned unwrapped.
func resolveDojoRoots(root string) (strategistRoot, basePath string, err error) {
	strategistRoot, basePath, err = cliutil.ResolveActiveBasePath(root)
	if err != nil {
		return "", "", fmt.Errorf("dojo: %w", err)
	}
	return strategistRoot, basePath, nil
}

func dojoItemLine(item domain.DojoCheckItem) string {
	if item.Passed {
		return fmt.Sprintf("  %s\t✓\n", item.Label)
	}
	detail := item.Detail
	if detail == "" {
		detail = "FAIL"
	}
	return fmt.Sprintf("  %s\t✗   ← %s\n", item.Label, detail)
}

func dojoSummaryLine(result domain.DojoCheckResult) string {
	if result.Passed() {
		return fmt.Sprintf("result\tPASS (%d checks)\n", len(result.Items))
	}
	return fmt.Sprintf("result\tFAIL (%d of %d checks failed)\n", result.FailCount(), len(result.Items))
}

func printDojoResult(result domain.DojoCheckResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	lines := []string{
		fmt.Sprintf("scenario\t%s\n", result.Scenario),
		"────────────────────────────────────────────────────\n",
	}
	for _, item := range result.Items {
		lines = append(lines, dojoItemLine(item))
	}
	lines = append(lines, "\n", dojoSummaryLine(result))
	for _, line := range lines {
		if _, err := fmt.Fprint(w, line); err != nil {
			return fmt.Errorf("dojo: write result: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("dojo: flush result: %w", err)
	}
	return nil
}

func init() {
	dojoCheckCmd.Flags().StringVar(&dojoRoot, "root", "", "path to .strategist/ root (default: .strategist)")
	dojoCheckCmd.Flags().BoolVar(&dojoFilesOnly, "files-only", false,
		"skip checks that require an emit.log run (emit_log, timing, pipeline)")
	dojoListCmd.Flags().StringVar(&dojoRoot, "root", "", "path to .strategist/ root (default: .strategist)")
	dojoCmd.AddCommand(dojoCheckCmd)
	dojoCmd.AddCommand(dojoListCmd)
	rootCmd.AddCommand(dojoCmd)
}

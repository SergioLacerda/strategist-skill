package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

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
	result := dojo.Run(criteria, basePath, strategistRoot, emitLogPath, dojoFilesOnly)

	if err := printDojoResult(result); err != nil {
		return err
	}
	if !result.Passed() {
		return fmt.Errorf("dojo check: scenario %q failed (%d checks failed)", scenario, result.FailCount())
	}
	return nil
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
	for _, e := range entries {
		if !e.IsDir() || e.Name() == ".last-run" {
			continue
		}
		criteriaPath := filepath.Join(dojoDir, e.Name(), "criteria.yaml")
		description := ""
		if raw, err := os.ReadFile(criteriaPath); err == nil {
			var c domain.DojoCriteria
			if yaml.Unmarshal(raw, &c) == nil {
				description = c.Description
			}
		}
		if _, err := fmt.Fprintf(w, "%s\t%s\n", e.Name(), description); err != nil {
			return fmt.Errorf("dojo list: write: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("dojo list: flush: %w", err)
	}
	return nil
}

func resolveDojoRoots(root string) (strategistRoot, basePath string, err error) {
	strategistRoot = root
	if strategistRoot == "" {
		strategistRoot = ".strategist"
	}

	raw, err := os.ReadFile(filepath.Join(strategistRoot, "active.yaml"))
	if err != nil {
		return "", "", fmt.Errorf("dojo: read active.yaml: %w", err)
	}
	var cfg domain.ActiveConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return "", "", fmt.Errorf("dojo: parse active.yaml: %w", err)
	}
	if cfg.BasePath == "" {
		return "", "", fmt.Errorf("dojo: active.yaml: base_path is empty")
	}
	basePath = cfg.BasePath
	if !filepath.IsAbs(basePath) {
		basePath = filepath.Join(filepath.Dir(strategistRoot), basePath)
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
	dojoCheckCmd.Flags().BoolVar(&dojoFilesOnly, "files-only", false, "skip emit_log validation")
	dojoListCmd.Flags().StringVar(&dojoRoot, "root", "", "path to .strategist/ root (default: .strategist)")
	dojoCmd.AddCommand(dojoCheckCmd)
	dojoCmd.AddCommand(dojoListCmd)
	rootCmd.AddCommand(dojoCmd)
}

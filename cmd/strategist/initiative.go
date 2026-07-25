package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var initiativeRoot string

var initiativeCmd = &cobra.Command{
	Use:   "initiative",
	Short: "Show mission slot providers and workspace status",
	Long: `Show the skill configured for each Strategist mission slot and the current workspace state.

SLOTS section reports, for each slot (discovery, refinement, execution):
  - provider id
  - canonical role and specialization class (rankeado or base)
  - whether the local provider manifest exists at .strategist/skills/<provider>/skill.yaml

WORKSPACE section reports:
  - mode and base_path from active.yaml
  - number of pending and done analysis cards
  - last recorded mission id (from memory/outcomes.jsonl, if present)`,
	RunE: runInitiative,
}

func runInitiative(_ *cobra.Command, _ []string) error {
	root, projectRoot, err := resolveInitiativeRoot()
	if err != nil {
		return err
	}

	cfg, err := readInitiativeConfig(root)
	if err != nil {
		return err
	}

	printStatusBanner("initiative")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

	if err := writeSlotsSection(w, cfg, root); err != nil {
		return err
	}
	if err := writeWorkspaceSection(w, cfg, root, projectRoot); err != nil {
		return err
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("initiative: flush output: %w", err)
	}
	return nil
}

func resolveInitiativeRoot() (string, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("initiative: get cwd: %w", err)
	}
	root, projectRoot, err := resolveStrategistRoot(initiativeRoot, cwd)
	if err != nil {
		return "", "", fmt.Errorf("initiative: %w", err)
	}
	return root, projectRoot, nil
}

func readInitiativeConfig(root string) (domain.ActiveConfig, error) {
	raw, err := os.ReadFile(filepath.Join(root, "active.yaml"))
	if err != nil {
		return domain.ActiveConfig{}, fmt.Errorf("initiative: read active.yaml: %w", err)
	}
	var cfg domain.ActiveConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return domain.ActiveConfig{}, fmt.Errorf("initiative: parse active.yaml: %w", err)
	}
	return cfg, nil
}

func writeSlotsSection(w *tabwriter.Writer, cfg domain.ActiveConfig, root string) error {
	if _, err := fmt.Fprintln(w, "SLOTS\t\t\t"); err != nil {
		return fmt.Errorf("initiative: write header: %w", err)
	}
	for _, slot := range []string{"discovery", "refinement", "execution"} {
		if err := writeSlotRow(w, cfg, root, slot); err != nil {
			return err
		}
	}
	return nil
}

func writeSlotRow(w *tabwriter.Writer, cfg domain.ActiveConfig, root, slot string) error {
	providerID := cfg.Slots[slot]
	role, class, status := providerRow(root, slot, providerID)
	if _, err := fmt.Fprintf(w, "%s\t%s\t%s %s\t%s\n", slot, providerID, role, class, status); err != nil {
		return fmt.Errorf("initiative: write row: %w", err)
	}
	return nil
}

func writeWorkspaceSection(w *tabwriter.Writer, cfg domain.ActiveConfig, root, projectRoot string) error {
	if _, err := fmt.Fprintln(w, "\t\t\t"); err != nil {
		return fmt.Errorf("initiative: write separator: %w", err)
	}
	if _, err := fmt.Fprintln(w, "WORKSPACE\t\t\t"); err != nil {
		return fmt.Errorf("initiative: write header: %w", err)
	}

	basePath := cfg.BasePath
	if basePath == "" {
		basePath = ".analysis"
	}
	absBase := basePath
	if !filepath.IsAbs(basePath) {
		absBase = filepath.Join(projectRoot, basePath)
	}

	pending := countEntries(filepath.Join(absBase, "pending"))
	done := countEntries(filepath.Join(absBase, "done"))
	lastMission := readLastMissionID(filepath.Join(root, "memory", "outcomes.jsonl"))

	rows := []struct{ k, v string }{
		{"mode", cfg.Mode},
		{"base_path", basePath},
		{"pending", formatCount(pending, "card")},
		{"done", formatCount(done, "missão")},
		{"last mission", lastMission},
	}
	for _, r := range rows {
		if _, err := fmt.Fprintf(w, "%s\t%s\t\t\n", r.k, r.v); err != nil {
			return fmt.Errorf("initiative: write row: %w", err)
		}
	}
	return nil
}

func init() {
	initiativeCmd.Flags().StringVar(&initiativeRoot, "root", "", "path to .strategist/ root (default: auto-discovered from CWD)")
	rootCmd.AddCommand(initiativeCmd)
}

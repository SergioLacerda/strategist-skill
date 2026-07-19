package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func formatCount(n int, unit string) string {
	if n < 0 {
		return "—"
	}
	plural := unit + "s"
	if unit == "missão" {
		plural = "missões"
	}
	if n == 1 {
		return fmt.Sprintf("1 %s", unit)
	}
	return fmt.Sprintf("%d %s", n, plural)
}

// countEntries returns the number of entries in dir, or -1 if dir is absent.
func countEntries(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return -1
	}
	return len(entries)
}

// readLastMissionID reads the last line of outcomes.jsonl and returns the mission_id field.
// Returns "—" if the file is absent or the field cannot be parsed.
func readLastMissionID(path string) string {
	f, err := os.Open(path) //nolint:gosec // G304: path derived from install root
	if err != nil {
		return "—"
	}
	defer func() { _ = f.Close() }() //nolint:errcheck // read-only file; close error is not actionable

	lastLine := lastNonEmptyLine(f)
	if lastLine == "" {
		return "—"
	}
	return missionIDFromJSONLine(lastLine)
}

func lastNonEmptyLine(f *os.File) string {
	var lastLine string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			lastLine = line
		}
	}
	return lastLine
}

func missionIDFromJSONLine(line string) string {
	var obj map[string]any
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		return "—"
	}
	if id, ok := obj["mission_id"].(string); ok && id != "" {
		return id
	}
	return "—"
}

func slotDefaultRole(slot string) string {
	switch slot {
	case "discovery":
		return "Ranger"
	case "refinement":
		return "Archivist"
	case "execution":
		return "Sniper"
	default:
		return strings.ToUpper(slot[:1]) + slot[1:]
	}
}

func canonicalRoleLabel(raw string) string {
	switch strings.ToLower(raw) {
	case "ranger":
		return "Ranger"
	case "archivist":
		return "Archivist"
	case "sniper":
		return "Sniper"
	default:
		if len(raw) == 0 {
			return raw
		}
		return strings.ToUpper(raw[:1]) + raw[1:]
	}
}

// providerRow derives the display columns for one slot row.
func providerRow(strategistDir, slot, providerID string) (role, class, manifestStatus string) {
	role = slotDefaultRole(slot)
	class = "(base)"
	manifestStatus = "⚠ manifest ausente"

	if providerID == "" {
		return
	}

	data, err := os.ReadFile(filepath.Join(strategistDir, "skills", providerID, "skill.yaml"))
	if err != nil {
		return
	}

	var m domain.ProviderManifest
	if yaml.Unmarshal(data, &m) != nil {
		return
	}

	if cr := m.SpecializationTaxonomy.CanonicalRole; cr != "" {
		role = canonicalRoleLabel(cr)
	}
	if m.ProviderClass == "rankeado" || m.SpecializationTaxonomy.ProviderClass == "rankeado" {
		class = "rankeado"
	}
	manifestStatus = "✓ manifest OK"
	return
}

func init() {
	initiativeCmd.Flags().StringVar(&initiativeRoot, "root", "", "path to .strategist/ root (default: auto-discovered from CWD)")
	rootCmd.AddCommand(initiativeCmd)
}

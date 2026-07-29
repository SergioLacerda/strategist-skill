package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/spf13/cobra"
)

var treasureChestDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Detect consistency drift across active.yaml, treasure-chests.yaml, and knowledge.index.yaml",
	Long: `Read-only diagnostic for treasure-chest add/remove consistency.

Reports chests present in one of the three registry truth layers
(active.yaml, treasure-chests.yaml, knowledge.index.yaml) but missing from
another — the divergence a batch write failure can leave behind.

Detection only: no automatic repair is attempted. Exits non-zero when any
divergence is found.`,
}

func init() {
	treasureChestDoctorCmd.RunE = runTreasureChestDoctor
	treasureChestCmd.AddCommand(treasureChestDoctorCmd)
}

func runTreasureChestDoctor(cmd *cobra.Command, _ []string) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}

	root, err := resolveTreasureChestActionRoot(cmd, "doctor")
	if err != nil {
		return err
	}
	rows, err := loadTreasureChestDoctorRows(root)
	if err != nil {
		return err
	}

	findings := diverging(rows)
	if len(findings) == 0 {
		fmt.Println("[Strategist] treasure-chest doctor: no consistency drift found across active.yaml, treasure-chests.yaml, knowledge.index.yaml")
		return nil
	}
	printTreasureChestDoctorFindings(findings)
	return fmt.Errorf("[Strategist] treasure-chest doctor: consistency drift in %d chest(s)", len(findings))
}

func loadTreasureChestDoctorRows(root string) ([]treasure.StatusRow, error) {
	active, err := treasure.LoadActiveChests(root)
	if err != nil {
		return nil, fmt.Errorf("treasure-chest doctor: %w", err)
	}
	governed, err := treasure.LoadGoverned(root)
	if err != nil {
		return nil, fmt.Errorf("treasure-chest doctor: %w", err)
	}
	indexed, err := treasure.LoadIndexed(root)
	if err != nil {
		return nil, fmt.Errorf("treasure-chest doctor: %w", err)
	}
	return treasure.MergeChestRows(active, governed, indexed, nil, nil), nil
}

func printTreasureChestDoctorFindings(findings []treasure.StatusRow) {
	for _, r := range findings {
		fmt.Fprintf(os.Stderr, "  ✗ %s: present in %s; absent from %s\n",
			r.ID, strings.Join(presentLayers(r), ", "), strings.Join(absentLayers(r), ", "))
	}
}

// diverging returns rows not present in all three registry truth layers.
func diverging(rows []treasure.StatusRow) []treasure.StatusRow {
	var out []treasure.StatusRow
	for _, r := range rows {
		if r.Configured && r.Governed && r.Indexed {
			continue
		}
		out = append(out, r)
	}
	return out
}

func presentLayers(r treasure.StatusRow) []string {
	var out []string
	if r.Configured {
		out = append(out, "active.yaml")
	}
	if r.Governed {
		out = append(out, "treasure-chests.yaml")
	}
	if r.Indexed {
		out = append(out, "knowledge.index.yaml")
	}
	return out
}

func absentLayers(r treasure.StatusRow) []string {
	var out []string
	if !r.Configured {
		out = append(out, "active.yaml")
	}
	if !r.Governed {
		out = append(out, "treasure-chests.yaml")
	}
	if !r.Indexed {
		out = append(out, "knowledge.index.yaml")
	}
	return out
}

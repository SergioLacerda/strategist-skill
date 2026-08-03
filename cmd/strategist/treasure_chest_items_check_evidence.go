package main

import (
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/spf13/cobra"
)

var treasureChestItemsCheckEvidenceCmd = &cobra.Command{
	Use:   "check-evidence <chest-id>",
	Short: "Advisory scan for expired, duplicate, or conflicting jewels in a chest",
	Long: `Runs three advisory checks over a chest's non-deprecated jewels:
  expired    — valid_until has passed
  duplicate  — same normalized statement (trimmed, case-folded)
  conflict   — overlapping source_refs with a differing statement or status

Never modifies a jewel. Always exits 0, regardless of findings — same
advisory-only posture as mission_quality, not a CI gate.`,
	Args: cobra.ExactArgs(1),
	RunE: itemCurationRunE("treasure-chest items check-evidence", func(_ *cobra.Command, root string, args []string) error {
		return runTreasureChestItemsCheckEvidence(root, args[0])
	}),
}

func runTreasureChestItemsCheckEvidence(root, chestID string) error {
	jewelsByChest, err := treasure.LoadJewels(root, bestEffortGoverned(root))
	if err != nil {
		return fmt.Errorf("treasure-chest items check-evidence: %w", err)
	}
	jewels := treasure.FilterJewels(jewelsByChest, treasure.JewelFilter{ChestID: chestID})
	report := treasure.CheckEvidenceQuality(chestID, jewels, time.Now())
	printEvidenceQualityReport(report)
	return nil
}

func printEvidenceQualityReport(r treasure.EvidenceQualityReport) {
	if !r.HasFindings() {
		fmt.Printf("[Strategist] treasure-chest items check-evidence %s: no findings\n", r.ChestID)
		return
	}
	for _, e := range r.Expired {
		fmt.Printf("[Strategist] expired: %s (valid_until: %s)\n", e.JewelID, e.ValidUntil)
	}
	for _, d := range r.Duplicates {
		fmt.Printf("[Strategist] duplicate: %s ~ %s\n", d.JewelIDA, d.JewelIDB)
	}
	for _, c := range r.Conflicts {
		fmt.Printf("[Strategist] conflict: %s <> %s (%s)\n", c.JewelIDA, c.JewelIDB, c.Reason)
	}
}

func init() {
	treasureChestItemsCmd.AddCommand(treasureChestItemsCheckEvidenceCmd)
}

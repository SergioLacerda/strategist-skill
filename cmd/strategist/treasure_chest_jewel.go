package main

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
)

// --- treasure-chest jewel (Track: treasure-chest-jewel-commands) ---
//
// `jewel` is the read-only inspection surface over all jewels regardless of status —
// unlike `mine --list`, which is scoped to the status:proposed curation queue only.
// See .analysis/pending/20260716-treasure-chest-jewel-commands-design.md.

var (
	treasureChestJewelStatus string
	treasureChestJewelChest  string
	treasureChestJewelFormat string
)

var treasureChestJewelCmd = &cobra.Command{
	Use:   "jewel",
	Short: "Inspect jewels regardless of curation status",
}

var treasureChestJewelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List jewels, optionally filtered by status or chest",
	Long: `Lists jewels across all chests.

Without --status: shows proposed, accepted, and verified jewels (everything "alive").
deprecated jewels are excluded unless --status=all or --status=deprecated is given.`,
	RunE: runTreasureChestJewelList,
}

func init() {
	treasureChestJewelListCmd.Flags().StringVar(&treasureChestJewelStatus, "status", "", "filter by status: all, proposed, accepted, verified, or deprecated (default: all except deprecated)")
	treasureChestJewelListCmd.Flags().StringVar(&treasureChestJewelChest, "chest", "", "filter by chest id")
	treasureChestJewelListCmd.Flags().StringVar(&treasureChestJewelFormat, "format", "table", "output format: table or json")

	treasureChestJewelCmd.AddCommand(treasureChestJewelListCmd)
	treasureChestCmd.AddCommand(treasureChestJewelCmd)
}

var validJewelListStatuses = map[string]bool{
	"all":                        true,
	domain.JewelStatusProposed:   true,
	domain.JewelStatusAccepted:   true,
	domain.JewelStatusVerified:   true,
	domain.JewelStatusDeprecated: true,
}

func runTreasureChestJewelList(cmd *cobra.Command, _ []string) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}

	if treasureChestJewelStatus != "" && !validJewelListStatuses[treasureChestJewelStatus] {
		return fmt.Errorf("treasure-chest jewel list: unknown --status %q (want all, proposed, accepted, verified, or deprecated)", treasureChestJewelStatus)
	}

	jewelsByChest, err := loadJewelsForCmd("treasure-chest jewel list")
	if err != nil {
		return err
	}

	filtered := filterAndSortJewels(jewelsByChest, treasureChestJewelChest, treasureChestJewelStatus)

	switch treasureChestJewelFormat {
	case "", "table":
		return renderJewelListTable(filtered)
	default:
		return fmt.Errorf("treasure-chest jewel list: unknown --format %q (want table or json)", treasureChestJewelFormat)
	}
}

// loadJewelsForCmd resolves the strategist root from cwd and loads all jewels,
// treating a missing/invalid governed-chests file as best-effort (not fatal) —
// governed data is only trust-ceiling context, not required for read-only inspection.
func loadJewelsForCmd(errPrefix string) (map[string][]jewelEntry, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("%s: get cwd: %w", errPrefix, err)
	}
	root, _, err := resolveStrategistRoot(treasureChestRoot, cwd)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errPrefix, err)
	}

	governed, govErr := loadGoverned(root)
	if govErr != nil {
		governed = nil
	}
	jewelsByChest, err := loadJewels(root, governed)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errPrefix, err)
	}
	return jewelsByChest, nil
}

func filterAndSortJewels(jewelsByChest map[string][]jewelEntry, chest, status string) []jewelEntry {
	var filtered []jewelEntry
	for _, list := range jewelsByChest {
		for _, j := range list {
			if jewelMatchesListFilter(j, chest, status) {
				filtered = append(filtered, j)
			}
		}
	}
	sort.Slice(filtered, func(i, k int) bool {
		if filtered[i].ChestID != filtered[k].ChestID {
			return filtered[i].ChestID < filtered[k].ChestID
		}
		return filtered[i].ID < filtered[k].ID
	})
	return filtered
}

func jewelMatchesListFilter(j jewelEntry, chest, status string) bool {
	if chest != "" && j.ChestID != chest {
		return false
	}
	switch status {
	case "":
		return j.Status != domain.JewelStatusDeprecated
	case "all":
		return true
	default:
		return j.Status == status
	}
}

func renderJewelListTable(jewels []jewelEntry) error {
	if len(jewels) == 0 {
		fmt.Println("[Strategist] treasure-chest jewel list: no jewels match the given filters")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "ID", "CHEST", "STATUS", "KIND", "TRUST", "SCORE", "STATEMENT"); err != nil {
		return fmt.Errorf("treasure-chest jewel list: write header: %w", err)
	}
	for _, j := range jewels {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%s\n", j.ID, j.ChestID, j.Status, j.Kind, j.Trust, j.Score.Value, j.Statement); err != nil {
			return fmt.Errorf("treasure-chest jewel list: write row: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("treasure-chest jewel list: flush: %w", err)
	}
	return nil
}

package main

import (
	"encoding/json"
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

var treasureChestJewelShowCmd = &cobra.Command{
	Use:   "show <jewel-id>",
	Short: "Show a single jewel's full content",
	Args:  cobra.ExactArgs(1),
	RunE:  runTreasureChestJewelShow,
}

func init() {
	treasureChestJewelListCmd.Flags().StringVar(&treasureChestJewelStatus, "status", "", "filter by status: all, proposed, accepted, verified, or deprecated (default: all except deprecated)")
	treasureChestJewelListCmd.Flags().StringVar(&treasureChestJewelChest, "chest", "", "filter by chest id")
	treasureChestJewelListCmd.Flags().StringVar(&treasureChestJewelFormat, "format", "table", "output format: table or json")

	treasureChestJewelShowCmd.Flags().StringVar(&treasureChestJewelFormat, "format", "table", "output format: table or json")

	treasureChestJewelCmd.AddCommand(treasureChestJewelListCmd)
	treasureChestJewelCmd.AddCommand(treasureChestJewelShowCmd)
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
	case "json":
		return renderJewelListJSON(filtered)
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

type jsonJewelListEntry struct {
	ID           string   `json:"id"`
	ChestID      string   `json:"chest_id"`
	Status       string   `json:"status"`
	Kind         string   `json:"kind"`
	Statement    string   `json:"statement"`
	SourceRefs   []string `json:"source_refs"`
	Trust        string   `json:"trust"`
	ScoreValue   int      `json:"score_value"`
	ScoreReasons []string `json:"score_reasons,omitempty"`
}

func renderJewelListJSON(jewels []jewelEntry) error {
	out := make([]jsonJewelListEntry, 0, len(jewels))
	for _, j := range jewels {
		out = append(out, jsonJewelListEntry{
			ID:           j.ID,
			ChestID:      j.ChestID,
			Status:       j.Status,
			Kind:         j.Kind,
			Statement:    j.Statement,
			SourceRefs:   j.SourceRefs,
			Trust:        j.Trust,
			ScoreValue:   j.Score.Value,
			ScoreReasons: j.Score.Reasons,
		})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("treasure-chest jewel list: encode json: %w", err)
	}
	return nil
}

func runTreasureChestJewelShow(cmd *cobra.Command, args []string) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}

	id := args[0]
	jewelsByChest, err := loadJewelsForCmd("treasure-chest jewel show")
	if err != nil {
		return err
	}

	var found *jewelEntry
	for _, list := range jewelsByChest {
		for i := range list {
			if list[i].ID == id {
				found = &list[i]
				break
			}
		}
	}
	if found == nil {
		return fmt.Errorf("treasure-chest jewel show: jewel %q not found", id)
	}

	switch treasureChestJewelFormat {
	case "", "table":
		return renderJewelShowTable(*found)
	case "json":
		return renderJewelShowJSON(*found)
	default:
		return fmt.Errorf("treasure-chest jewel show: unknown --format %q (want table or json)", treasureChestJewelFormat)
	}
}

func renderJewelShowTable(j jewelEntry) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	rows := [][2]string{
		{"id", j.ID},
		{"chest_id", j.ChestID},
		{"kind", j.Kind},
		{"status", j.Status},
		{"trust", j.Trust},
		{"statement", j.Statement},
		{"source_refs", fmt.Sprintf("%v", j.SourceRefs)},
		{"reviewed_by", j.ReviewedBy},
		{"last_reviewed", j.LastReviewed},
		{"score.value", fmt.Sprintf("%d", j.Score.Value)},
		{"score.reasons", fmt.Sprintf("%v", j.Score.Reasons)},
		{"applicability.scope", fmt.Sprintf("%v", j.Applicability.Scope)},
		{"applicability.applies_when", fmt.Sprintf("%v", j.Applicability.AppliesWhen)},
		{"applicability.avoid_when", fmt.Sprintf("%v", j.Applicability.AvoidWhen)},
		{"verification.evidence_refs", fmt.Sprintf("%v", j.Verification.EvidenceRefs)},
	}
	for _, r := range rows {
		if _, err := fmt.Fprintf(w, "%s\t%s\n", r[0], r[1]); err != nil {
			return fmt.Errorf("treasure-chest jewel show: write row: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("treasure-chest jewel show: flush: %w", err)
	}
	return nil
}

type jsonJewelShowEntry struct {
	ID            string             `json:"id"`
	ChestID       string             `json:"chest_id"`
	Kind          string             `json:"kind"`
	Status        string             `json:"status"`
	Trust         string             `json:"trust"`
	Statement     string             `json:"statement"`
	SourceRefs    []string           `json:"source_refs"`
	ReviewedBy    string             `json:"reviewed_by"`
	LastReviewed  string             `json:"last_reviewed,omitempty"`
	Score         jewelScore         `json:"score"`
	Applicability jewelApplicability `json:"applicability"`
	Verification  jewelVerification  `json:"verification"`
}

func renderJewelShowJSON(j jewelEntry) error {
	out := jsonJewelShowEntry{
		ID:            j.ID,
		ChestID:       j.ChestID,
		Kind:          j.Kind,
		Status:        j.Status,
		Trust:         j.Trust,
		Statement:     j.Statement,
		SourceRefs:    j.SourceRefs,
		ReviewedBy:    j.ReviewedBy,
		LastReviewed:  j.LastReviewed,
		Score:         j.Score,
		Applicability: j.Applicability,
		Verification:  j.Verification,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("treasure-chest jewel show: encode json: %w", err)
	}
	return nil
}

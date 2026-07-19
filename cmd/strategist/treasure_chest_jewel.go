package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/spf13/cobra"
)

// --- treasure-chest jewel (Track: treasure-chest-jewel-commands) ---
//
// `jewel` is the read-only inspection surface over all jewels regardless of status —
// unlike `mine --list`, which is scoped to the status:proposed curation queue only.
// See .analysis/pending/20260716-treasure-chest-jewel-commands-design.md.

type treasureChestJewelListOptions struct {
	Status string
	Chest  string
	Format string
}

type treasureChestJewelShowOptions struct {
	Format string
}

type treasureChestJewelVerifyOptions struct {
	Evidence string
}

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
}

var treasureChestJewelShowCmd = &cobra.Command{
	Use:   "show <jewel-id>",
	Short: "Show a single jewel's full content",
	Args:  cobra.ExactArgs(1),
}

// jewelCurationRunE wraps a curation action with the shared telemetry-silencing and
// strategist-root resolution boilerplate every jewel mutation subcommand needs.
func jewelCurationRunE(errPrefix string, action func(cmd *cobra.Command, root string, args []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if run := telemetryRunFromCmd(cmd); run != nil {
			run.SetSilent()
		}
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("%s: get cwd: %w", errPrefix, err)
		}
		root, _, err := resolveStrategistRoot(treasureChestRootFromCmd(cmd), cwd)
		if err != nil {
			return fmt.Errorf("%s: %w", errPrefix, err)
		}
		return action(cmd, root, args)
	}
}

var treasureChestJewelAcceptCmd = &cobra.Command{
	Use:   "accept <jewel-id>...",
	Short: "Promote jewel ids to status: accepted",
	Args:  cobra.MinimumNArgs(1),
	RunE: jewelCurationRunE("treasure-chest jewel accept", func(_ *cobra.Command, root string, args []string) error {
		return runTreasureChestMinePromote(root, treasure.ParseJewelIDs(args...), domain.JewelStatusAccepted, "")
	}),
}

var treasureChestJewelVerifyCmd = &cobra.Command{
	Use:   "verify <jewel-id>...",
	Short: "Promote jewel ids to status: verified (requires --evidence)",
	Args:  cobra.MinimumNArgs(1),
}

var treasureChestJewelDeprecateCmd = &cobra.Command{
	Use:   "deprecate <jewel-id>...",
	Short: "Mark jewel ids as status: deprecated",
	Args:  cobra.MinimumNArgs(1),
	RunE: jewelCurationRunE("treasure-chest jewel deprecate", func(_ *cobra.Command, root string, args []string) error {
		return runTreasureChestMinePromote(root, treasure.ParseJewelIDs(args...), domain.JewelStatusDeprecated, "")
	}),
}

var treasureChestJewelMigrateStatusCmd = &cobra.Command{
	Use:   "migrate-status",
	Short: "One-time migration: rewrite legacy status: active jewels to status: accepted",
	Args:  cobra.ExactArgs(0),
	RunE: jewelCurationRunE("treasure-chest jewel migrate-status", func(_ *cobra.Command, root string, _ []string) error {
		return runTreasureChestMineMigrateStatus(root)
	}),
}

func init() {
	listOpts := treasureChestJewelListOptions{Format: "table"}
	showOpts := treasureChestJewelShowOptions{Format: "table"}
	verifyOpts := treasureChestJewelVerifyOptions{}
	treasureChestJewelListCmd.Flags().StringVar(&listOpts.Status, "status", "", "filter by status: all, proposed, accepted, verified, or deprecated (default: all except deprecated)")
	treasureChestJewelListCmd.Flags().StringVar(&listOpts.Chest, "chest", "", "filter by chest id")
	treasureChestJewelListCmd.Flags().StringVar(&listOpts.Format, "format", "table", "output format: table or json")
	treasureChestJewelListCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runTreasureChestJewelList(cmd, args, listOpts)
	}

	treasureChestJewelShowCmd.Flags().StringVar(&showOpts.Format, "format", "table", "output format: table or json")
	treasureChestJewelShowCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runTreasureChestJewelShow(cmd, args, showOpts)
	}

	treasureChestJewelVerifyCmd.Flags().StringVar(&verifyOpts.Evidence, "evidence", "", "evidence reference recorded with verify")
	treasureChestJewelVerifyCmd.RunE = jewelCurationRunE("treasure-chest jewel verify", func(cmd *cobra.Command, root string, args []string) error {
		evidence := stringFlag(cmd, "evidence", verifyOpts.Evidence)
		if evidence == "" {
			return fmt.Errorf("treasure-chest jewel verify: --evidence is required")
		}
		return runTreasureChestMinePromote(root, treasure.ParseJewelIDs(args...), domain.JewelStatusVerified, evidence)
	})

	treasureChestJewelCmd.AddCommand(treasureChestJewelListCmd)
	treasureChestJewelCmd.AddCommand(treasureChestJewelShowCmd)
	treasureChestJewelCmd.AddCommand(treasureChestJewelAcceptCmd)
	treasureChestJewelCmd.AddCommand(treasureChestJewelVerifyCmd)
	treasureChestJewelCmd.AddCommand(treasureChestJewelDeprecateCmd)
	treasureChestJewelCmd.AddCommand(treasureChestJewelMigrateStatusCmd)

	treasureChestCmd.AddCommand(treasureChestJewelCmd)
}

var validJewelListStatuses = map[string]bool{
	"all":                        true,
	domain.JewelStatusProposed:   true,
	domain.JewelStatusAccepted:   true,
	domain.JewelStatusVerified:   true,
	domain.JewelStatusDeprecated: true,
}

func runTreasureChestJewelList(cmd *cobra.Command, _ []string, opts treasureChestJewelListOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	opts.Status = stringFlag(cmd, "status", opts.Status)
	opts.Chest = stringFlag(cmd, "chest", opts.Chest)
	opts.Format = stringFlag(cmd, "format", opts.Format)

	if opts.Status != "" && !validJewelListStatuses[opts.Status] {
		return fmt.Errorf("treasure-chest jewel list: unknown --status %q (want all, proposed, accepted, verified, or deprecated)", opts.Status)
	}

	jewelsByChest, err := loadJewelsForCmd(cmd, "treasure-chest jewel list")
	if err != nil {
		return err
	}

	filtered := treasure.FilterJewels(jewelsByChest, treasure.JewelFilter{
		ChestID: opts.Chest,
		Status:  opts.Status,
	})

	switch opts.Format {
	case "", "table":
		return renderJewelListTable(filtered)
	case "json":
		return renderJewelListJSON(filtered)
	default:
		return fmt.Errorf("treasure-chest jewel list: unknown --format %q (want table or json)", opts.Format)
	}
}

// loadJewelsForCmd resolves the strategist root from cwd and loads all jewels,
// treating a missing/invalid governed-chests file as best-effort (not fatal) —
// governed data is only trust-ceiling context, not required for read-only inspection.
func loadJewelsForCmd(cmd *cobra.Command, errPrefix string) (map[string][]treasure.Jewel, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("%s: get cwd: %w", errPrefix, err)
	}
	root, _, err := resolveStrategistRoot(treasureChestRootFromCmd(cmd), cwd)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errPrefix, err)
	}

	governed, govErr := treasure.LoadGoverned(root)
	if govErr != nil {
		governed = nil
	}
	jewelsByChest, err := treasure.LoadJewels(root, governed)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errPrefix, err)
	}
	return jewelsByChest, nil
}

func renderJewelListTable(jewels []treasure.Jewel) error {
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

func renderJewelListJSON(jewels []treasure.Jewel) error {
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

func runTreasureChestJewelShow(cmd *cobra.Command, args []string, opts treasureChestJewelShowOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	opts.Format = stringFlag(cmd, "format", opts.Format)

	id := args[0]
	jewelsByChest, err := loadJewelsForCmd(cmd, "treasure-chest jewel show")
	if err != nil {
		return err
	}

	found, ok := treasure.FindJewel(jewelsByChest, id)
	if !ok {
		return fmt.Errorf("treasure-chest jewel show: jewel %q not found", id)
	}

	switch opts.Format {
	case "", "table":
		return renderJewelShowTable(found)
	case "json":
		return renderJewelShowJSON(found)
	default:
		return fmt.Errorf("treasure-chest jewel show: unknown --format %q (want table or json)", opts.Format)
	}
}

func renderJewelShowTable(j treasure.Jewel) error {
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
		{"applicability.Scope", fmt.Sprintf("%v", j.Applicability.Scope)},
		{"applicability.applies_when", fmt.Sprintf("%v", j.Applicability.AppliesWhen)},
		{"applicability.avoid_when", fmt.Sprintf("%v", j.Applicability.AvoidWhen)},
		{"verification.evidence_refs", fmt.Sprintf("%v", j.Verification.EvidenceRefs)},
		{"history", fmt.Sprintf("%v", j.History)},
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
	ID            string                       `json:"id"`
	ChestID       string                       `json:"chest_id"`
	Kind          string                       `json:"kind"`
	Status        string                       `json:"status"`
	Trust         string                       `json:"trust"`
	Statement     string                       `json:"statement"`
	SourceRefs    []string                     `json:"source_refs"`
	ReviewedBy    string                       `json:"reviewed_by"`
	LastReviewed  string                       `json:"last_reviewed,omitempty"`
	Score         treasure.JewelScore          `json:"score"`
	Applicability treasure.JewelApplicability  `json:"applicability"`
	Verification  treasure.JewelVerification   `json:"verification"`
	History       []treasure.JewelHistoryEntry `json:"history,omitempty"`
}

func renderJewelShowJSON(j treasure.Jewel) error {
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
		History:       j.History,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("treasure-chest jewel show: encode json: %w", err)
	}
	return nil
}

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/spf13/cobra"
)

// --- treasure-chest mine (Track: treasure-chest-index-mine-pipeline) ---
//
// `mine` is the human curation command over status:proposed jewels: list, accept, verify
// (with evidence), or deprecate. Non-interactive by design so it can be scripted. Also
// carries the one-time active -> accepted migration (--migrate-status; see ADR 0012). See
// .analysis/refined/treasure-chest-index-mine-pipeline/design.md.

type treasureChestMineOptions struct {
	List          bool
	Format        string
	Accept        string
	Verify        string
	Evidence      string
	Deprecate     string
	MigrateStatus bool
}

var treasureChestMineCmd = &cobra.Command{
	Use:   "mine",
	Short: "Curate proposed jewels: list, accept, verify, or deprecate",
	Long: `Human curation over status:proposed jewels produced by "strategist treasure-chest index".

Non-interactive flags (exactly one required):
  --list                         list status:proposed jewels awaiting curation
  --accept <jewel-id>             promote a jewel to status: accepted
  --verify <jewel-id> --evidence  promote a jewel to status: verified (evidence required)
  --deprecate <jewel-id>          mark a jewel as status: deprecated
	--migrate-status                one-time migration: legacy status: active -> accepted`,
	Deprecated: "use 'strategist treasure-chest jewel' subcommands instead",
}

func init() {
	opts := treasureChestMineOptions{Format: "table"}
	treasureChestMineCmd.Flags().BoolVar(&opts.List, "list", false, "list status:proposed jewels awaiting curation")
	treasureChestMineCmd.Flags().StringVar(&opts.Format, "format", "table", "output format for --list: table or json")
	treasureChestMineCmd.Flags().StringVar(&opts.Accept, "accept", "", "promote a jewel id to status: accepted")
	treasureChestMineCmd.Flags().StringVar(&opts.Verify, "verify", "", "promote a jewel id to status: verified (requires --evidence)")
	treasureChestMineCmd.Flags().StringVar(&opts.Evidence, "evidence", "", "evidence reference recorded with --verify")
	treasureChestMineCmd.Flags().StringVar(&opts.Deprecate, "deprecate", "", "mark a jewel id as status: deprecated")
	treasureChestMineCmd.Flags().BoolVar(&opts.MigrateStatus, "migrate-status", false, "one-time migration: rewrite legacy status: active jewels to status: accepted")
	treasureChestMineCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runTreasureChestMine(cmd, args, opts)
	}
	treasureChestCmd.AddCommand(treasureChestMineCmd)
}

func runTreasureChestMine(cmd *cobra.Command, _ []string, opts treasureChestMineOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	opts = treasureChestMineOptionsFromFlags(cmd, opts)
	if err := validateTreasureChestMineOptions(opts); err != nil {
		return err
	}

	root, err := resolveTreasureChestCommandRoot(cmd, "mine")
	if err != nil {
		return err
	}
	return runTreasureChestMineAction(root, opts)
}

func treasureChestMineOptionsFromFlags(cmd *cobra.Command, opts treasureChestMineOptions) treasureChestMineOptions {
	opts.List = boolFlag(cmd, "list", opts.List)
	opts.Format = stringFlag(cmd, "format", opts.Format)
	opts.Accept = stringFlag(cmd, "accept", opts.Accept)
	opts.Verify = stringFlag(cmd, "verify", opts.Verify)
	opts.Evidence = stringFlag(cmd, "evidence", opts.Evidence)
	opts.Deprecate = stringFlag(cmd, "deprecate", opts.Deprecate)
	opts.MigrateStatus = boolFlag(cmd, "migrate-status", opts.MigrateStatus)
	return opts
}

func validateTreasureChestMineOptions(opts treasureChestMineOptions) error {
	if mineActionCount(opts) != 1 {
		return fmt.Errorf("treasure-chest mine: specify exactly one of --list, --accept, --verify, --deprecate, --migrate-status")
	}
	if opts.Verify != "" && opts.Evidence == "" {
		return fmt.Errorf("treasure-chest mine: --verify requires --evidence")
	}
	return nil
}

func mineActionCount(opts treasureChestMineOptions) int {
	actions := 0
	for _, set := range []bool{opts.List, opts.Accept != "", opts.Verify != "", opts.Deprecate != "", opts.MigrateStatus} {
		if set {
			actions++
		}
	}
	return actions
}

func runTreasureChestMineAction(root string, opts treasureChestMineOptions) error {
	switch {
	case opts.List:
		return runTreasureChestMineList(root, opts.Format)
	case opts.Accept != "":
		return runTreasureChestMinePromote(root, treasure.ParseJewelIDs(opts.Accept), domain.JewelStatusAccepted, "")
	case opts.Verify != "":
		return runTreasureChestMinePromote(root, treasure.ParseJewelIDs(opts.Verify), domain.JewelStatusVerified, opts.Evidence)
	case opts.Deprecate != "":
		return runTreasureChestMinePromote(root, treasure.ParseJewelIDs(opts.Deprecate), domain.JewelStatusDeprecated, "")
	default:
		return runTreasureChestMineMigrateStatus(root)
	}
}

// --- list ---

func runTreasureChestMineList(root, format string) error {
	// Best-effort: governed is trust-ceiling context only, not required for listing. A
	// corrupt/missing treasure-chests.yaml just means trust ceilings go unchecked here.
	governed, govErr := treasure.LoadGoverned(root)
	if govErr != nil {
		governed = nil
	}
	jewelsByChest, err := treasure.LoadJewels(root, governed)
	if err != nil {
		return fmt.Errorf("treasure-chest mine: %w", err)
	}

	proposed := treasure.ProposedJewels(jewelsByChest)

	switch format {
	case "", "table":
		return renderMineTable(proposed)
	case "json":
		return renderMineJSON(proposed)
	default:
		return fmt.Errorf("treasure-chest mine: unknown --format %q (want table or json)", format)
	}
}

func renderMineTable(jewels []treasure.Jewel) error {
	if len(jewels) == 0 {
		fmt.Println("[Strategist] treasure-chest mine: no proposed jewels awaiting curation")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", "ID", "CHEST", "KIND", "TRUST", "SCORE", "STATEMENT"); err != nil {
		return fmt.Errorf("treasure-chest mine: write header: %w", err)
	}
	for _, j := range jewels {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\n", j.ID, j.ChestID, j.Kind, j.Trust, j.Score.Value, j.Statement); err != nil {
			return fmt.Errorf("treasure-chest mine: write row: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("treasure-chest mine: flush: %w", err)
	}
	return nil
}

type jsonMineJewel struct {
	ID           string   `json:"id"`
	ChestID      string   `json:"chest_id"`
	Kind         string   `json:"kind"`
	Statement    string   `json:"statement"`
	SourceRefs   []string `json:"source_refs"`
	Trust        string   `json:"trust"`
	Status       string   `json:"status"`
	ScoreValue   int      `json:"score_value"`
	ScoreReasons []string `json:"score_reasons,omitempty"`
	Scope        []string `json:"scope,omitempty"`
}

func renderMineJSON(jewels []treasure.Jewel) error {
	out := make([]jsonMineJewel, 0, len(jewels))
	for _, j := range jewels {
		out = append(out, jsonMineJewel{
			ID:           j.ID,
			ChestID:      j.ChestID,
			Kind:         j.Kind,
			Statement:    j.Statement,
			SourceRefs:   j.SourceRefs,
			Trust:        j.Trust,
			Status:       j.Status,
			ScoreValue:   j.Score.Value,
			ScoreReasons: j.Score.Reasons,
			Scope:        j.Applicability.Scope,
		})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("treasure-chest mine: encode json: %w", err)
	}
	return nil
}

// --- accept / verify / deprecate ---

// runTreasureChestMinePromote sets jewel statuses via yaml.Node mutation, preserving
// comments/structure. A deprecated jewel can only be re-deprecated (idempotent no-op path),
// never promoted back to accepted/verified — deprecation is intentionally sticky.
func runTreasureChestMinePromote(root string, ids []string, newStatus, evidenceRef string) error {
	if len(ids) == 0 {
		return fmt.Errorf("treasure-chest mine: provide at least one jewel id")
	}
	reviewedAt := time.Now()
	for _, id := range ids {
		if err := treasure.PromoteJewel(root, id, newStatus, evidenceRef, reviewedAt); err != nil {
			return fmt.Errorf("treasure-chest mine: promote %q: %w", id, err)
		}
		fmt.Printf("[Strategist] treasure-chest mine: jewel %s -> status: %s\n", id, newStatus)
	}
	return nil
}

// --- migration (Track: active -> accepted, see ADR 0012) ---

// runTreasureChestMineMigrateStatus rewrites every jewels.yaml entry with the removed
// legacy status: active to status: accepted. This is the explicit one-time migration path
// referenced by ValidateJewelStatus's error message; after running it, treasure.LoadJewels rejects
// any remaining "active" entries as drift, not a silently-tolerated fallback.
func runTreasureChestMineMigrateStatus(root string) error {
	migrated, err := treasure.MigrateLegacyJewelStatus(root)
	if err != nil {
		return fmt.Errorf("treasure-chest mine: %w", err)
	}
	if migrated == 0 {
		fmt.Println(`[Strategist] treasure-chest mine --migrate-status: no legacy "active" jewels found, nothing to migrate`)
		return nil
	}
	fmt.Printf("[Strategist] treasure-chest mine --migrate-status: %d jewel(s) migrated active -> accepted\n", migrated)
	return nil
}

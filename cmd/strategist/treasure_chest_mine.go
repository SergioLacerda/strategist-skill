package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// --- treasure-chest mine (Track: treasure-chest-index-mine-pipeline) ---
//
// `mine` is the human curation command over status:proposed jewels: list, accept, verify
// (with evidence), or deprecate. Non-interactive by design so it can be scripted. Also
// carries the one-time active -> accepted migration (--migrate-status; see ADR 0012). See
// .analysis/refined/treasure-chest-index-mine-pipeline/design.md.

var (
	treasureChestMineList          bool
	treasureChestMineFormat        string
	treasureChestMineAccept        string
	treasureChestMineVerify        string
	treasureChestMineEvidence      string
	treasureChestMineDeprecate     string
	treasureChestMineMigrateStatus bool
)

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
	RunE: runTreasureChestMine,
}

func init() {
	treasureChestMineCmd.Flags().BoolVar(&treasureChestMineList, "list", false, "list status:proposed jewels awaiting curation")
	treasureChestMineCmd.Flags().StringVar(&treasureChestMineFormat, "format", "table", "output format for --list: table or json")
	treasureChestMineCmd.Flags().StringVar(&treasureChestMineAccept, "accept", "", "promote a jewel id to status: accepted")
	treasureChestMineCmd.Flags().StringVar(&treasureChestMineVerify, "verify", "", "promote a jewel id to status: verified (requires --evidence)")
	treasureChestMineCmd.Flags().StringVar(&treasureChestMineEvidence, "evidence", "", "evidence reference recorded with --verify")
	treasureChestMineCmd.Flags().StringVar(&treasureChestMineDeprecate, "deprecate", "", "mark a jewel id as status: deprecated")
	treasureChestMineCmd.Flags().BoolVar(&treasureChestMineMigrateStatus, "migrate-status", false, "one-time migration: rewrite legacy status: active jewels to status: accepted")
	treasureChestCmd.AddCommand(treasureChestMineCmd)
}

func runTreasureChestMine(cmd *cobra.Command, _ []string) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}

	actions := 0
	for _, set := range []bool{
		treasureChestMineList,
		treasureChestMineAccept != "",
		treasureChestMineVerify != "",
		treasureChestMineDeprecate != "",
		treasureChestMineMigrateStatus,
	} {
		if set {
			actions++
		}
	}
	if actions != 1 {
		return fmt.Errorf("treasure-chest mine: specify exactly one of --list, --accept, --verify, --deprecate, --migrate-status")
	}
	if treasureChestMineVerify != "" && treasureChestMineEvidence == "" {
		return fmt.Errorf("treasure-chest mine: --verify requires --evidence")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("treasure-chest mine: get cwd: %w", err)
	}
	root, _, err := resolveStrategistRoot(treasureChestRoot, cwd)
	if err != nil {
		return fmt.Errorf("treasure-chest mine: %w", err)
	}

	switch {
	case treasureChestMineList:
		return runTreasureChestMineList(root)
	case treasureChestMineAccept != "":
		return runTreasureChestMinePromote(root, treasureChestMineAccept, domain.JewelStatusAccepted, "")
	case treasureChestMineVerify != "":
		return runTreasureChestMinePromote(root, treasureChestMineVerify, domain.JewelStatusVerified, treasureChestMineEvidence)
	case treasureChestMineDeprecate != "":
		return runTreasureChestMinePromote(root, treasureChestMineDeprecate, domain.JewelStatusDeprecated, "")
	default:
		return runTreasureChestMineMigrateStatus(root)
	}
}

// --- list ---

func runTreasureChestMineList(root string) error {
	// Best-effort: governed is trust-ceiling context only, not required for listing. A
	// corrupt/missing treasure-chests.yaml just means trust ceilings go unchecked here.
	governed, govErr := loadGoverned(root)
	if govErr != nil {
		governed = nil
	}
	jewelsByChest, err := loadJewels(root, governed)
	if err != nil {
		return fmt.Errorf("treasure-chest mine: %w", err)
	}

	var proposed []jewelEntry
	for _, list := range jewelsByChest {
		for _, j := range list {
			if j.Status == domain.JewelStatusProposed {
				proposed = append(proposed, j)
			}
		}
	}
	sort.Slice(proposed, func(i, k int) bool {
		if proposed[i].ChestID != proposed[k].ChestID {
			return proposed[i].ChestID < proposed[k].ChestID
		}
		return proposed[i].ID < proposed[k].ID
	})

	switch treasureChestMineFormat {
	case "", "table":
		return renderMineTable(proposed)
	case "json":
		return renderMineJSON(proposed)
	default:
		return fmt.Errorf("treasure-chest mine: unknown --format %q (want table or json)", treasureChestMineFormat)
	}
}

func renderMineTable(jewels []jewelEntry) error {
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

func renderMineJSON(jewels []jewelEntry) error {
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

// runTreasureChestMinePromote sets a jewel's status via yaml.Node mutation, preserving
// comments/structure. A deprecated jewel can only be re-deprecated (idempotent no-op path),
// never promoted back to accepted/verified — deprecation is intentionally sticky.
func runTreasureChestMinePromote(root, id, newStatus, evidenceRef string) error {
	path := filepath.Join(root, "jewels.yaml")
	doc, err := readYAMLNode(path)
	if err != nil {
		return fmt.Errorf("treasure-chest mine: %w", err)
	}
	entry, err := findJewelEntry(doc, id)
	if err != nil {
		return fmt.Errorf("treasure-chest mine: %w", err)
	}

	current := ""
	if v := mappingValue(entry, "status"); v != nil {
		current = v.Value
	}
	if current == domain.JewelStatusDeprecated && newStatus != domain.JewelStatusDeprecated {
		return fmt.Errorf("treasure-chest mine: jewel %q is deprecated, cannot promote to %s", id, newStatus)
	}

	setMappingField(entry, "status", newStatus)
	setMappingField(entry, "reviewed_by", "human")
	setMappingField(entry, "last_reviewed", time.Now().UTC().Format("2006-01-02"))
	if newStatus == domain.JewelStatusVerified {
		appendEvidenceRef(entry, evidenceRef)
	}

	if _, err := writeYAMLNodes(yamlWrite{path: path, doc: doc}); err != nil {
		return fmt.Errorf("treasure-chest mine: %w", err)
	}
	fmt.Printf("[Strategist] treasure-chest mine: jewel %s -> status: %s\n", id, newStatus)
	return nil
}

// --- migration (Track: active -> accepted, see ADR 0012) ---

// runTreasureChestMineMigrateStatus rewrites every jewels.yaml entry with the removed
// legacy status: active to status: accepted. This is the explicit one-time migration path
// referenced by ValidateJewelStatus's error message; after running it, loadJewels rejects
// any remaining "active" entries as drift, not a silently-tolerated fallback.
func runTreasureChestMineMigrateStatus(root string) error {
	path := filepath.Join(root, "jewels.yaml")
	doc, err := readYAMLNode(path)
	if err != nil {
		return fmt.Errorf("treasure-chest mine: %w", err)
	}
	mapping, err := rootMapping(doc)
	if err != nil {
		return fmt.Errorf("treasure-chest mine: %w", err)
	}

	seq := mappingValue(mapping, "jewels")
	migrated := 0
	if seq != nil {
		for _, entry := range seq.Content {
			if entry.Kind != yaml.MappingNode {
				continue
			}
			v := mappingValue(entry, "status")
			if v != nil && v.Value == "active" {
				setMappingField(entry, "status", domain.JewelStatusAccepted)
				migrated++
			}
		}
	}

	if migrated == 0 {
		fmt.Println(`[Strategist] treasure-chest mine --migrate-status: no legacy "active" jewels found, nothing to migrate`)
		return nil
	}
	if _, err := writeYAMLNodes(yamlWrite{path: path, doc: doc}); err != nil {
		return fmt.Errorf("treasure-chest mine: %w", err)
	}
	fmt.Printf("[Strategist] treasure-chest mine --migrate-status: %d jewel(s) migrated active -> accepted\n", migrated)
	return nil
}

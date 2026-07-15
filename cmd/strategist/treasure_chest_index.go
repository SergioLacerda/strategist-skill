package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// --- treasure-chest index (Track: treasure-chest-index-mine-pipeline) ---
//
// `index` is the primary public command for the offline organization plane: it folds the
// internal `scan` phase (mission-history clustering/gap-mining), detects and polishes
// reusable knowledge units into deduplicated status:proposed jewels, and refreshes the
// compiled knowledge index. See
// .analysis/refined/treasure-chest-index-mine-pipeline/design.md.

// missionHistoryChestID is the virtual chest_id used for jewels generated from mission
// history (scan-derived clusters/gaps), which are not tied to any single treasure chest.
const missionHistoryChestID = "mission-history"

var treasureChestIndexCmd = &cobra.Command{
	Use:   "index",
	Short: "Rebuild the offline knowledge substrate (scan, propose jewels, refresh compiled index)",
	Long: `Rebuild the offline knowledge substrate from declared sources and mission history.

Responsibilities:
  - Validate configured, governed, indexed, and compiled layers.
  - Run the internal scan phase over <base_path>/refined/**/tasks.md and <base_path>/done/**.
  - Detect and polish reusable knowledge units into deduplicated status:proposed jewels.
  - Rebuild the compiled knowledge index (.strategist/.compiled/.index.gz).

Use --include-historical to opt in to indexing T2/T3 sources into the compiled artifact.`,
	RunE: runTreasureChestIndexCmd,
}

func init() {
	treasureChestIndexCmd.Flags().BoolVar(&treasureChestIncludeHistorical, "include-historical", false, "include T2/T3 historical sources in the compiled index rebuild")
	treasureChestCmd.AddCommand(treasureChestIndexCmd)
}

func runTreasureChestIndexCmd(cmd *cobra.Command, _ []string) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("treasure-chest index: get cwd: %w", err)
	}
	root, _, err := resolveStrategistRoot(treasureChestRoot, cwd)
	if err != nil {
		return fmt.Errorf("treasure-chest index: %w", err)
	}

	governed, err := loadGoverned(root)
	if err != nil {
		return fmt.Errorf("treasure-chest index: %w", err)
	}

	_, basePath, err := resolveDojoRoots(root)
	if err != nil {
		return fmt.Errorf("treasure-chest index: %w", err)
	}

	missions, err := scanMissions(basePath)
	if err != nil {
		return fmt.Errorf("treasure-chest index: %w", err)
	}
	clusters := buildClusters(missions)
	gaps := buildGaps(missions)

	clustersDir := filepath.Join(root, "treasure", "clusters")
	gapsDir := filepath.Join(root, "treasure", "gaps")
	if err := writeScanOutputs(clustersDir, clusters, gapsDir, gaps); err != nil {
		return fmt.Errorf("treasure-chest index: %w", err)
	}

	candidates := buildJewelCandidates(clusters, gaps)
	written, skipped, err := writeProposedJewels(root, candidates)
	if err != nil {
		return fmt.Errorf("treasure-chest index: %w", err)
	}

	if !treasureChestIncludeHistorical {
		if historical := historicalCount(governedRows(governed)); historical > 0 {
			fmt.Printf("[Strategist] treasure-chest index: %d historical/lower-trust source(s) excluded from default indexing.\n", historical)
			fmt.Println("             Use --include-historical to opt in.")
		}
	}

	indexPath := filepath.Join(root, "knowledge.index.yaml")
	c := compile.Compiler{}
	if err := c.CompileAll(root, indexPath); err != nil {
		return fmt.Errorf("treasure-chest index: %w", err)
	}

	fmt.Printf("[Strategist] treasure-chest index: %d mission(s) scanned, %d candidate(s) found, "+
		"%d proposed jewel(s) written, %d duplicate(s) skipped, compiled artifact refreshed\n",
		len(missions), len(candidates), written, skipped)
	return nil
}

// governedRows adapts the governed-chest map into the minimal []chestRow shape
// historicalCount needs (trustTier only).
func governedRows(governed map[string]govChest) []chestRow {
	rows := make([]chestRow, 0, len(governed))
	for _, gc := range governed {
		rows = append(rows, chestRow{trustTier: gc.Trust.Tier})
	}
	return rows
}

// --- candidate detection / polishing ---

// buildJewelCandidates polishes scan-derived clusters and gaps into proposed jewel
// candidates. Candidates are not tied to a specific treasure chest — they use the virtual
// missionHistoryChestID — since they are derived from mission history, not chest content.
// Polishing here is deterministic (id, statement, source_refs, score) since Go code has no
// LLM access; richer LLM-governed polishing (classification nuance, applicability text) may
// refine these candidates in a future pass, but must never promote them beyond proposed.
func buildJewelCandidates(clusters []cluster, gaps []gap) []jewelEntry {
	candidates := make([]jewelEntry, 0, len(clusters)+len(gaps))
	for _, c := range clusters {
		candidates = append(candidates, jewelEntry{
			ID:         "jewel-" + c.ID,
			ChestID:    missionHistoryChestID,
			Kind:       "pattern",
			Statement:  fmt.Sprintf("Recurring theme across missions: %s", strings.Join(c.Tags, ", ")),
			SourceRefs: missionSourceRefs(c.CitedMissions),
			Trust:      "T2",
			Status:     domain.JewelStatusProposed,
			ReviewedBy: "agent",
			Score: jewelScore{
				Value: clusterCandidateScore(c),
				Reasons: []string{
					fmt.Sprintf("recurring across %d missions", len(c.CitedMissions)),
					"shared tags: " + strings.Join(c.Tags, ", "),
				},
			},
			Applicability: jewelApplicability{Scope: []string{"all"}},
		})
	}
	for _, g := range gaps {
		candidates = append(candidates, jewelEntry{
			ID:         "jewel-gap-" + g.ID,
			ChestID:    missionHistoryChestID,
			Kind:       "gap",
			Statement:  fmt.Sprintf("Open side quest %s still pending", g.ID),
			SourceRefs: missionSourceRefs(g.CitedMissions),
			Trust:      "T2",
			Status:     domain.JewelStatusProposed,
			ReviewedBy: "agent",
			Score: jewelScore{
				Value:   gapCandidateScore(g),
				Reasons: []string{fmt.Sprintf("still pending in %d mission(s)", len(g.CitedMissions))},
			},
			Applicability: jewelApplicability{Scope: []string{"all"}},
		})
	}
	return candidates
}

func missionSourceRefs(missionIDs []string) []string {
	refs := make([]string, 0, len(missionIDs))
	for _, id := range missionIDs {
		refs = append(refs, missionHistoryChestID+"#"+id)
	}
	return refs
}

// clusterCandidateScore is a deterministic 0-100 candidate quality signal — never a
// promotion authority, only a ranking/economy hint (see jewel_generation contract).
func clusterCandidateScore(c cluster) int {
	v := 40 + len(c.CitedMissions)*10 + len(c.Tags)*5
	if v > 100 {
		v = 100
	}
	return v
}

func gapCandidateScore(g gap) int {
	v := 30 + len(g.CitedMissions)*15
	if v > 100 {
		v = 100
	}
	return v
}

// --- dedup + write ---

// writeProposedJewels appends candidates not already present (by id) in jewels.yaml,
// preserving existing comments/structure via yaml.Node. It returns how many were written
// vs. skipped as duplicates. A candidate with an id that already exists — proposed or
// otherwise — is always skipped, never overwritten: index must not clobber human curation.
func writeProposedJewels(root string, candidates []jewelEntry) (written, skipped int, err error) {
	if len(candidates) == 0 {
		return 0, 0, nil
	}

	path := filepath.Join(root, "jewels.yaml")
	doc, err := readYAMLNode(path)
	if err != nil {
		return 0, 0, fmt.Errorf("index proposed jewels: %w", err)
	}
	mapping, err := rootMapping(doc)
	if err != nil {
		return 0, 0, fmt.Errorf("index proposed jewels: %w", err)
	}
	seq := findOrCreateSequence(mapping, "jewels")

	for _, cand := range candidates {
		if _, idx := findEntryByID(seq, cand.ID); idx != -1 {
			skipped++
			continue
		}
		var entry yaml.Node
		if err := entry.Encode(cand); err != nil {
			return written, skipped, fmt.Errorf("index proposed jewels: encode %s: %w", cand.ID, err)
		}
		seq.Content = append(seq.Content, &entry)
		written++
	}

	if written == 0 {
		return written, skipped, nil
	}
	if _, err := writeYAMLNodes(yamlWrite{path: path, doc: doc}); err != nil {
		return written, skipped, fmt.Errorf("index proposed jewels: %w", err)
	}
	return written, skipped, nil
}

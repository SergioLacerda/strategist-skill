package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/spf13/cobra"
)

// --- treasure-chest index (Track: treasure-chest-index-mine-pipeline) ---
//
// `index` is the primary public command for the offline organization plane: it folds the
// internal `scan` phase (mission-history clustering/gap-mining), detects and polishes
// reusable knowledge units into deduplicated status:proposed jewels, and refreshes the
// compiled knowledge index. See
// .analysis/refined/treasure-chest-index-mine-pipeline/design.md.

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
}

func init() {
	var opts treasureChestIndexOptions
	treasureChestIndexCmd.Flags().BoolVar(&opts.IncludeHistorical, "include-historical", false, "include T2/T3 historical sources in the compiled index rebuild")
	treasureChestIndexCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runTreasureChestIndexCmd(cmd, args, opts)
	}
	treasureChestCmd.AddCommand(treasureChestIndexCmd)
}

type treasureChestIndexOptions struct {
	IncludeHistorical bool
}

func runTreasureChestIndexCmd(cmd *cobra.Command, _ []string, opts treasureChestIndexOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	opts.IncludeHistorical = boolFlag(cmd, "include-historical", opts.IncludeHistorical)

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("treasure-chest index: get cwd: %w", err)
	}
	root, _, err := resolveStrategistRoot(treasureChestRootFromCmd(cmd), cwd)
	if err != nil {
		return fmt.Errorf("treasure-chest index: %w", err)
	}

	governed, err := treasure.LoadGoverned(root)
	if err != nil {
		return fmt.Errorf("treasure-chest index: %w", err)
	}
	scoringPolicy, err := treasure.LoadScoringPolicy(root)
	if err != nil {
		return fmt.Errorf("treasure-chest index: %w", err)
	}

	_, basePath, err := resolveDojoRoots(root)
	if err != nil {
		return fmt.Errorf("treasure-chest index: %w", err)
	}

	missions, err := treasure.ScanMissions(basePath)
	if err != nil {
		return fmt.Errorf("treasure-chest index: %w", err)
	}
	clusters := treasure.BuildClusters(missions)
	gaps := treasure.BuildGaps(missions)

	clustersDir := filepath.Join(root, "treasure", "clusters")
	gapsDir := filepath.Join(root, "treasure", "gaps")
	if err := treasure.WriteScanOutputs(clustersDir, clusters, gapsDir, gaps); err != nil {
		return fmt.Errorf("treasure-chest index: %w", err)
	}

	candidates := treasure.BuildJewelCandidatesWithPolicy(clusters, gaps, scoringPolicy)
	written, skipped, err := treasure.WriteProposedJewels(root, candidates)
	if err != nil {
		return fmt.Errorf("treasure-chest index: %w", err)
	}

	if !opts.IncludeHistorical {
		if historical := treasure.HistoricalCount(governedRows(governed)); historical > 0 {
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

// governedRows adapts the governed-chest map into the minimal []treasure.StatusRow shape
// historicalCount needs (trustTier only).
func governedRows(governed map[string]treasure.GovernedChest) []treasure.StatusRow {
	rows := make([]treasure.StatusRow, 0, len(governed))
	for _, gc := range governed {
		rows = append(rows, treasure.StatusRow{TrustTier: gc.Trust.Tier})
	}
	return rows
}

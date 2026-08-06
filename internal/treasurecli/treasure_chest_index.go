package treasurecli

import (
	"fmt"
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
	treasureChestIndexCmd.Flags().BoolVar(&opts.IncludeHistorical, flagIncludeHistorical, false, "include T2/T3 historical sources in the compiled index rebuild")
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
	opts.IncludeHistorical = boolFlag(cmd, flagIncludeHistorical, opts.IncludeHistorical)

	root, err := resolveTreasureChestActionRoot(cmd, "index")
	if err != nil {
		return err
	}

	jewelPhase, err := runTreasureChestIndexJewelPhase(cmd, root)
	if err != nil {
		return err
	}

	potionPhase, err := runTreasureChestIndexPotionPhase(root, jewelPhase.governed)
	if err != nil {
		return err
	}

	if err := finalizeTreasureChestIndex(root, jewelPhase.governed, opts.IncludeHistorical); err != nil {
		return err
	}

	fmt.Printf("[Strategist] treasure-chest index: %d mission(s) scanned, %d candidate(s) found, "+
		"%d proposed jewel(s) written, %d duplicate(s) skipped, "+
		"%d proposed potion(s) written, %d duplicate(s) skipped, compiled artifact refreshed\n",
		len(jewelPhase.missions), len(jewelPhase.candidates), jewelPhase.written, jewelPhase.skipped,
		potionPhase.written, potionPhase.skipped)
	return nil
}

// indexJewelPhaseResult carries the jewel-mining phase's outputs forward — governed is
// reused by the potion phase and the historical-source warning; the rest feeds the
// final summary line.
type indexJewelPhaseResult struct {
	governed   map[string]treasure.GovernedChest
	missions   []treasure.ScannedMission
	candidates []treasure.Jewel
	written    int
	skipped    int
}

// runTreasureChestIndexJewelPhase runs the mission-history scan and proposes jewel
// candidates from it — the original `index` behavior, unchanged by the Potion/chest-
// content scan extension in runTreasureChestIndexPotionPhase.
func runTreasureChestIndexJewelPhase(cmd *cobra.Command, root string) (indexJewelPhaseResult, error) {
	governed, err := treasure.LoadGoverned(root)
	if err != nil {
		return indexJewelPhaseResult{}, fmt.Errorf("treasure-chest index: %w", err)
	}
	scoringPolicy, err := treasure.LoadScoringPolicy(root)
	if err != nil {
		return indexJewelPhaseResult{}, fmt.Errorf("treasure-chest index: %w", err)
	}

	_, basePath, err := resolveDojoRoots(root)
	if err != nil {
		return indexJewelPhaseResult{}, fmt.Errorf("treasure-chest index: %w", err)
	}
	scanResult, err := treasure.RunScanPipeline(root, basePath)
	if err != nil {
		return indexJewelPhaseResult{}, fmt.Errorf("treasure-chest index: %w", err)
	}
	printTreasureChestIndexWarnings(cmd, scanResult.Warnings)

	candidates := treasure.BuildJewelCandidatesWithPolicy(scanResult.Clusters, scanResult.Gaps, scoringPolicy)
	written, skipped, err := treasure.WriteProposedJewels(root, candidates)
	if err != nil {
		return indexJewelPhaseResult{}, fmt.Errorf("treasure-chest index: %w", err)
	}

	return indexJewelPhaseResult{
		governed:   governed,
		missions:   scanResult.Missions,
		candidates: candidates,
		written:    written,
		skipped:    skipped,
	}, nil
}

type indexPotionPhaseResult struct {
	written int
	skipped int
}

// runTreasureChestIndexPotionPhase scans registered chests' own content (ask #1 /
// SQ-001) and proposes Potion candidates from it.
func runTreasureChestIndexPotionPhase(root string, governed map[string]treasure.GovernedChest) (indexPotionPhaseResult, error) {
	candidates, err := scanRegisteredChestsForPotions(root, governed)
	if err != nil {
		return indexPotionPhaseResult{}, err
	}
	written, skipped, err := treasure.WriteProposedPotions(root, candidates)
	if err != nil {
		return indexPotionPhaseResult{}, fmt.Errorf("treasure-chest index: %w", err)
	}
	return indexPotionPhaseResult{written: written, skipped: skipped}, nil
}

// finalizeTreasureChestIndex prints the historical-source warning (if applicable) and
// rebuilds the compiled knowledge index.
func finalizeTreasureChestIndex(root string, governed map[string]treasure.GovernedChest, includeHistorical bool) error {
	printHistoricalIndexWarning(governed, includeHistorical)

	indexPath := filepath.Join(root, "knowledge.index.yaml")
	c := compile.Compiler{}
	if err := c.CompileAll(root, indexPath); err != nil {
		return fmt.Errorf("treasure-chest index: %w", err)
	}
	return nil
}

func printTreasureChestIndexWarnings(cmd *cobra.Command, warnings []treasure.ScanWarning) {
	for _, warning := range warnings {
		cmd.PrintErrf("Warning: treasure-chest index: skipped inconsistent mission file: %v\n", warning)
	}
}

func printHistoricalIndexWarning(governed map[string]treasure.GovernedChest, includeHistorical bool) {
	if includeHistorical {
		return
	}
	if historical := treasure.HistoricalCount(governedRows(governed)); historical > 0 {
		fmt.Printf("[Strategist] treasure-chest index: %d historical/lower-trust source(s) excluded from default indexing.\n", historical)
		fmt.Println("             Use --include-historical to opt in.")
	}
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

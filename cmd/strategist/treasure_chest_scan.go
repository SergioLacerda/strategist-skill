package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/spf13/cobra"
)

var treasureChestScanCmd = &cobra.Command{
	Use:    "scan",
	Hidden: true,
	Short:  "[internal] Mine .analysis/refined and .analysis/done for recurring clusters and open gaps",
	Long: `Internal phase folded into "strategist treasure-chest index" — not part of the
public steady-state UX. Kept as a standalone command for debugging/dry-run inspection of
the scan phase in isolation.

Scan mission history for recurring themes and unresolved side quests.

Input: <base_path>/refined/**/tasks.md and <base_path>/done/** only. Never reads
<base_path>/pending/ or <base_path>/archived/ reports.

Method: lexical/tag matching only. No embeddings, no vector index.

Output: .strategist/treasure/clusters/ and .strategist/treasure/gaps/, regenerated from
scratch on every run (safe to delete — see docs/configuration.md § Storage Domain).

Use --dry-run to preview output without writing to disk.`,
}

func init() {
	var opts treasureChestScanOptions
	treasureChestScanCmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "print what would be written without touching disk")
	treasureChestScanCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runTreasureChestScan(cmd, args, opts)
	}
	treasureChestCmd.AddCommand(treasureChestScanCmd)
}

type treasureChestScanOptions struct {
	DryRun bool
}

func runTreasureChestScan(cmd *cobra.Command, _ []string, opts treasureChestScanOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	opts.DryRun = boolFlag(cmd, "dry-run", opts.DryRun)

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("treasure-chest scan: get cwd: %w", err)
	}
	root, _, err := resolveStrategistRoot(treasureChestRootFromCmd(cmd), cwd)
	if err != nil {
		return fmt.Errorf("treasure-chest scan: %w", err)
	}
	_, basePath, err := resolveDojoRoots(root)
	if err != nil {
		return fmt.Errorf("treasure-chest scan: %w", err)
	}

	missions, err := treasure.ScanMissions(basePath)
	if err != nil {
		return fmt.Errorf("treasure-chest scan: %w", err)
	}

	clusters := treasure.BuildClusters(missions)
	gaps := treasure.BuildGaps(missions)

	clustersDir := filepath.Join(root, "treasure", "clusters")
	gapsDir := filepath.Join(root, "treasure", "gaps")

	if opts.DryRun {
		printScanDryRun(missions, clusters, gaps, clustersDir, gapsDir)
		return nil
	}

	if err := treasure.WriteScanOutputs(clustersDir, clusters, gapsDir, gaps); err != nil {
		return fmt.Errorf("treasure-chest scan: %w", err)
	}

	fmt.Printf("[Strategist] treasure-chest scan: %d mission(s) scanned, %d cluster(s) and %d gap(s) written\n",
		len(missions), len(clusters), len(gaps))
	return nil
}

func printScanDryRun(missions []treasure.ScannedMission, clusters []treasure.Cluster, gaps []treasure.Gap, clustersDir, gapsDir string) {
	fmt.Printf("[Strategist] treasure-chest scan (dry-run): %d mission(s) scanned, would write %d cluster(s), %d gap(s)\n",
		len(missions), len(clusters), len(gaps))
	for _, c := range clusters {
		fmt.Printf("  cluster: %s\n", filepath.Join(clustersDir, c.ID+".md"))
	}
	for _, g := range gaps {
		fmt.Printf("  gap: %s\n", filepath.Join(gapsDir, g.ID+".md"))
	}
}

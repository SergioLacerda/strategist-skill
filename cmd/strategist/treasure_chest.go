package main

import (
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/spf13/cobra"
)

// --- command flags ---

type treasureChestOptions struct {
	Root              string
	DoIndex           bool
	IncludeHistorical bool
	Format            string
	Scope             string
}

var treasureChestCmd = &cobra.Command{
	Use:   "treasure-chest",
	Short: "Show treasure chest runtime status and index health",
	Long: `Show the current treasure chest configuration, governance, index, and compiled artifact status.

Default: read-only status view. Merges four runtime truth layers:
  - .strategist/active.yaml         (configured/scoped chests)
  - .strategist/treasure-chests.yaml (governed trust and routing policy)
  - .strategist/knowledge.index.yaml (indexed retrieval sources)
  - .strategist/.compiled/.index.gz  (compiled fast-path artifact)

Use --index to rebuild the compiled knowledge index from declared sources.
Use --include-historical with --index to opt in to indexing T2/T3 sources.`,
}

func runTreasureChest(cmd *cobra.Command, _ []string, opts treasureChestOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	opts = treasureChestOptionsFromFlags(cmd, opts)
	root, err := resolveTreasureChestActionRoot(cmd, "treasure-chest")
	if err != nil {
		return err
	}

	// Load the four truth layers (each is best-effort; only active.yaml is mandatory).
	activeChests, err := treasure.LoadActiveChests(root)
	if err != nil {
		return fmt.Errorf("treasure-chest: %w", err)
	}

	governed, govErr := treasure.LoadGoverned(root)
	indexed, idxErr := treasure.LoadIndexed(root)
	compiledIDs, compiledAt, compErr := treasure.LoadCompiledIndex(root)
	jewels, jewelErr := treasure.LoadJewels(root, governed)
	if jewelErr != nil {
		return fmt.Errorf("treasure-chest: %w", jewelErr)
	}

	rows := treasure.MergeChestRows(activeChests, governed, indexed, compiledIDs, jewels)

	if opts.DoIndex {
		return runTreasureChestIndex(root, rows, opts.IncludeHistorical)
	}

	rows = treasure.FilterRowsByScope(rows, opts.Scope)

	switch opts.Format {
	case "", outputFormatTable:
		// fall through to table rendering below
	case outputFormatJSON:
		return renderTreasureChestJSON(os.Stdout, root, rows, compErr, govErr, idxErr, compiledAt)
	default:
		return fmt.Errorf("treasure-chest: unknown --format %q (want table or json)", opts.Format)
	}

	return renderTreasureChestTable(root, rows, govErr, idxErr, compErr, compiledAt)
}

func treasureChestOptionsFromFlags(cmd *cobra.Command, opts treasureChestOptions) treasureChestOptions {
	opts.Root = stringFlag(cmd, flagRoot, opts.Root)
	opts.DoIndex = boolFlag(cmd, flagIndex, opts.DoIndex)
	opts.IncludeHistorical = boolFlag(cmd, flagIncludeHistorical, opts.IncludeHistorical)
	opts.Format = stringFlag(cmd, flagFormat, opts.Format)
	opts.Scope = stringFlag(cmd, "scope", opts.Scope)
	return opts
}

func init() {
	opts := treasureChestOptions{Format: outputFormatTable}
	treasureChestCmd.PersistentFlags().StringVar(&opts.Root, flagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	treasureChestCmd.Flags().BoolVar(&opts.DoIndex, flagIndex, false, "rebuild compiled knowledge index from declared sources")
	treasureChestCmd.Flags().BoolVar(&opts.IncludeHistorical, flagIncludeHistorical, false, "include T2/T3 historical sources in index rebuild (requires --index)")
	treasureChestCmd.Flags().StringVar(&opts.Format, flagFormat, outputFormatTable, "output format: table or json")
	treasureChestCmd.Flags().StringVar(&opts.Scope, "scope", "", "filter output by slot scope (e.g. discovery, refinement, execution)")
	treasureChestCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runTreasureChest(cmd, args, opts)
	}
	rootCmd.AddCommand(treasureChestCmd)
}

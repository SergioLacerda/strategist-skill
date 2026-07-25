package main

import (
	"fmt"

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
		root, err := resolveTreasureChestActionRoot(cmd, errPrefix)
		if err != nil {
			return err
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
	listOpts := treasureChestJewelListOptions{Format: outputFormatTable}
	showOpts := treasureChestJewelShowOptions{Format: outputFormatTable}
	verifyOpts := treasureChestJewelVerifyOptions{}
	treasureChestJewelListCmd.Flags().StringVar(&listOpts.Status, "status", "", "filter by status: all, proposed, accepted, verified, or deprecated (default: all except deprecated)")
	treasureChestJewelListCmd.Flags().StringVar(&listOpts.Chest, "chest", "", "filter by chest id")
	treasureChestJewelListCmd.Flags().StringVar(&listOpts.Format, flagFormat, outputFormatTable, "output format: table or json")
	treasureChestJewelListCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runTreasureChestJewelList(cmd, args, listOpts)
	}

	treasureChestJewelShowCmd.Flags().StringVar(&showOpts.Format, flagFormat, outputFormatTable, "output format: table or json")
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

// loadJewelsForCmd resolves the strategist root from cwd and loads all jewels,
// treating a missing/invalid governed-chests file as best-effort (not fatal) —
// governed data is only trust-ceiling context, not required for read-only inspection.
func loadJewelsForCmd(cmd *cobra.Command, errPrefix string) (map[string][]treasure.Jewel, error) {
	root, err := resolveTreasureChestActionRoot(cmd, errPrefix)
	if err != nil {
		return nil, err
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

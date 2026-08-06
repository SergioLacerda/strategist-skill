package treasurecli

import (
	"errors"
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/spf13/cobra"
)

// --- treasure-chest items (Track: treasure-chest-cli-unification) ---
//
// `items` is the unified CRUD surface over both Jewel and Potion rows — the two
// candidate types a treasure chest can hold. It replaces `jewel` (renamed) and
// absorbs `mine` (removed; see Decisions D2/D5 in
// .analysis/refined/treasure-chest-cli-unification/analysis.md). Jewels and Potions
// are never hand-created via `add`: they are proposed by `index` (status: proposed)
// and then curated by a human via accept/verify/deprecate — the same tombstone
// doctrine already used by `treasure-chest remove` for chests.

type treasureChestItemsListOptions struct {
	Kind   string
	Status string
	Chest  string
	Format string
}

type treasureChestItemsShowOptions struct {
	Format string
}

type treasureChestItemsVerifyOptions struct {
	Evidence string
}

var treasureChestItemsCmd = &cobra.Command{
	Use:   "items",
	Short: "Inspect and curate jewels and potions regardless of curation status",
}

var treasureChestItemsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List items (jewels and/or potions), optionally filtered by kind, status, or chest",
	Long: `Lists items across all chests.

Without --status: shows proposed, accepted, and verified items (everything "alive").
deprecated items are excluded unless --status=all or --status=deprecated is given.
Without --kind: shows both jewels and potions.`,
}

var treasureChestItemsShowCmd = &cobra.Command{
	Use:   "show <item-id>",
	Short: "Show a single item's (jewel or potion) full content",
	Args:  cobra.ExactArgs(1),
}

// itemCurationRunE wraps a curation action with the shared telemetry-silencing and
// strategist-root resolution boilerplate every items mutation subcommand needs.
func itemCurationRunE(errPrefix string, action func(cmd *cobra.Command, root string, args []string) error) func(*cobra.Command, []string) error {
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

var treasureChestItemsAcceptCmd = &cobra.Command{
	Use:   "accept <item-id>...",
	Short: "Promote item ids to status: accepted",
	Args:  cobra.MinimumNArgs(1),
	RunE: itemCurationRunE("treasure-chest items accept", func(_ *cobra.Command, root string, args []string) error {
		return runTreasureChestItemsPromote(root, treasure.ParseItemIDs(args...), domain.JewelStatusAccepted, "")
	}),
}

var treasureChestItemsVerifyCmd = &cobra.Command{
	Use:   "verify <item-id>...",
	Short: "Promote item ids to status: verified (requires --evidence)",
	Args:  cobra.MinimumNArgs(1),
}

var treasureChestItemsDeprecateCmd = &cobra.Command{
	Use:   "deprecate <item-id>...",
	Short: "Mark item ids as status: deprecated",
	Args:  cobra.MinimumNArgs(1),
	RunE: itemCurationRunE("treasure-chest items deprecate", func(_ *cobra.Command, root string, args []string) error {
		return runTreasureChestItemsPromote(root, treasure.ParseItemIDs(args...), domain.JewelStatusDeprecated, "")
	}),
}

var treasureChestItemsMigrateStatusCmd = &cobra.Command{
	Use:   "migrate-status",
	Short: "One-time migration: rewrite legacy status: active jewels to status: accepted",
	Args:  cobra.ExactArgs(0),
	RunE: itemCurationRunE("treasure-chest items migrate-status", func(_ *cobra.Command, root string, _ []string) error {
		return runTreasureChestItemsMigrateStatus(root)
	}),
}

func init() {
	listOpts := treasureChestItemsListOptions{Format: outputFormatTable}
	showOpts := treasureChestItemsShowOptions{Format: outputFormatTable}
	verifyOpts := treasureChestItemsVerifyOptions{}
	treasureChestItemsListCmd.Flags().StringVar(&listOpts.Kind, "kind", "", "filter by item kind: jewel or potion (default: both)")
	treasureChestItemsListCmd.Flags().StringVar(&listOpts.Status, "status", "", "filter by status: all, proposed, accepted, verified, or deprecated (default: all except deprecated)")
	treasureChestItemsListCmd.Flags().StringVar(&listOpts.Chest, "chest", "", "filter by chest id")
	treasureChestItemsListCmd.Flags().StringVar(&listOpts.Format, flagFormat, outputFormatTable, "output format: table or json")
	treasureChestItemsListCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runTreasureChestItemsList(cmd, args, listOpts)
	}

	treasureChestItemsShowCmd.Flags().StringVar(&showOpts.Format, flagFormat, outputFormatTable, "output format: table or json")
	treasureChestItemsShowCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runTreasureChestItemsShow(cmd, args, showOpts)
	}

	treasureChestItemsVerifyCmd.Flags().StringVar(&verifyOpts.Evidence, "evidence", "", "evidence reference recorded with verify (jewels only — see PromotePotion)")
	treasureChestItemsVerifyCmd.RunE = itemCurationRunE("treasure-chest items verify", func(cmd *cobra.Command, root string, args []string) error {
		evidence := stringFlag(cmd, "evidence", verifyOpts.Evidence)
		if evidence == "" {
			return fmt.Errorf("treasure-chest items verify: --evidence is required")
		}
		return runTreasureChestItemsPromote(root, treasure.ParseItemIDs(args...), domain.JewelStatusVerified, evidence)
	})

	treasureChestItemsCmd.AddCommand(treasureChestItemsListCmd)
	treasureChestItemsCmd.AddCommand(treasureChestItemsShowCmd)
	treasureChestItemsCmd.AddCommand(treasureChestItemsAcceptCmd)
	treasureChestItemsCmd.AddCommand(treasureChestItemsVerifyCmd)
	treasureChestItemsCmd.AddCommand(treasureChestItemsDeprecateCmd)
	treasureChestItemsCmd.AddCommand(treasureChestItemsMigrateStatusCmd)

	treasureChestCmd.AddCommand(treasureChestItemsCmd)
}

// loadJewelsForCmd resolves the strategist root from cwd and loads all jewels,
// treating a missing/invalid governed-chests file as best-effort (not fatal) —
// governed data is only trust-ceiling context, not required for read-only inspection.
func loadJewelsForCmd(cmd *cobra.Command, errPrefix string) (map[string][]treasure.Jewel, error) {
	root, err := resolveTreasureChestActionRoot(cmd, errPrefix)
	if err != nil {
		return nil, err
	}
	jewelsByChest, err := treasure.LoadJewels(root, bestEffortGoverned(root))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errPrefix, err)
	}
	return jewelsByChest, nil
}

// loadPotionsForCmd is loadJewelsForCmd's Potion counterpart.
func loadPotionsForCmd(cmd *cobra.Command, errPrefix string) (map[string][]treasure.Potion, error) {
	root, err := resolveTreasureChestActionRoot(cmd, errPrefix)
	if err != nil {
		return nil, err
	}
	potionsByChest, err := treasure.LoadPotions(root, bestEffortGoverned(root))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errPrefix, err)
	}
	return potionsByChest, nil
}

func bestEffortGoverned(root string) map[string]treasure.GovernedChest {
	governed, err := treasure.LoadGoverned(root)
	if err != nil {
		return nil
	}
	return governed
}

// --- accept / verify / deprecate ---

// runTreasureChestItemsPromote sets item statuses via yaml.Node mutation, preserving
// comments/structure. Each id is tried as a jewel first, then as a potion — the two
// manifests share the id namespace by convention, never both. A deprecated item can
// only be re-deprecated (idempotent no-op path), never promoted back to
// accepted/verified — deprecation is intentionally sticky (see PromoteJewel/PromotePotion).
func runTreasureChestItemsPromote(root string, ids []string, newStatus, evidenceRef string) error {
	if len(ids) == 0 {
		return fmt.Errorf("treasure-chest items: provide at least one item id")
	}
	reviewedAt := time.Now()
	for _, id := range ids {
		if err := promoteItem(root, id, newStatus, evidenceRef, reviewedAt); err != nil {
			return fmt.Errorf("treasure-chest items: promote %q: %w", id, err)
		}
		fmt.Printf("[Strategist] treasure-chest items: %s -> status: %s\n", id, newStatus)
	}
	return nil
}

func promoteItem(root, id, newStatus, evidenceRef string, reviewedAt time.Time) error {
	if err := treasure.PromoteJewel(root, id, newStatus, evidenceRef, reviewedAt); err == nil {
		return nil
	} else if !errors.Is(err, treasure.ErrJewelNotFound) {
		return fmt.Errorf("promote jewel: %w", err)
	}

	if err := treasure.PromotePotion(root, id, newStatus, evidenceRef, reviewedAt); err == nil {
		return nil
	} else if !errors.Is(err, treasure.ErrPotionNotFound) {
		return fmt.Errorf("promote potion: %w", err)
	}

	return fmt.Errorf("item %q not found (checked jewels and potions)", id)
}

// --- migration (Track: active -> accepted, see ADR 0012) ---

// runTreasureChestItemsMigrateStatus rewrites every jewels.yaml entry with the removed
// legacy status: active to status: accepted. Jewel-only: Potion never had a legacy
// "active" status (its schema was introduced after ADR 0012's migration).
func runTreasureChestItemsMigrateStatus(root string) error {
	migrated, err := treasure.MigrateLegacyJewelStatus(root)
	if err != nil {
		return fmt.Errorf("treasure-chest items: %w", err)
	}
	if migrated == 0 {
		fmt.Println(`[Strategist] treasure-chest items --migrate-status: no legacy "active" jewels found, nothing to migrate`)
		return nil
	}
	fmt.Printf("[Strategist] treasure-chest items --migrate-status: %d jewel(s) migrated active -> accepted\n", migrated)
	return nil
}

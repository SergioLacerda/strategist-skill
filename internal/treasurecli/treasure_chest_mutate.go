package treasurecli

import (
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/spf13/cobra"
)

// --- SQ-006 (Track T-I): treasure-chest add / remove ---
//
// Implements the command contract documented in docs/cli-reference.md § treasure-chest
// (Planned: treasure-chest add / treasure-chest remove). Mutations are applied via
// yaml.Node round-tripping so existing comments/formatting in active.yaml,
// treasure-chests.yaml, and knowledge.index.yaml survive the edit — every other writer
// in this codebase either struct-marshals (losing comments) or does an install-time-only
// placeholder replace (internal/install/active_yaml.go), neither of which is safe against
// an already-populated file.

type treasureChestAddOptions struct {
	ID         string
	Scope      string
	TrustTier  string
	ReviewedBy string
	Tags       string
	IndexAfter bool
}

type treasureChestRemoveOptions struct {
	ID string
}

var treasureChestAddCmd = &cobra.Command{
	Use:   "add <path>",
	Short: "Register a new treasure chest across active/governed/indexed layers",
	Long: `Add a treasure chest source, updating active.yaml, treasure-chests.yaml, and
knowledge.index.yaml together in one pass. Existing file comments/formatting are preserved.

Leaves the compiled index (.compiled/.index.gz) stale unless --index is also passed.`,
	Args: cobra.ExactArgs(1),
}

var treasureChestRemoveCmd = &cobra.Command{
	Use:   "remove [path]",
	Short: "Tombstone a treasure chest (mark inactive, do not hard-delete)",
	Long: `Remove a treasure chest from active.yaml's active declarations and mark it
status: inactive in treasure-chests.yaml and knowledge.index.yaml, preserving an audit
trail instead of hard-deleting the entry. Resolve by positional path or --id; if both are
given and disagree, the command stops with an ambiguity error.`,
	Args: cobra.MaximumNArgs(1),
}

func init() {
	addOpts := treasureChestAddOptions{Scope: "all", TrustTier: "T1", ReviewedBy: "human"}
	removeOpts := treasureChestRemoveOptions{}
	treasureChestAddCmd.Flags().StringVar(&addOpts.ID, "id", "", "chest id (default: derived from the last path segment)")
	treasureChestAddCmd.Flags().StringVar(&addOpts.Scope, "scope", "all", "slot Scope: all, discovery, refinement, or execution")
	treasureChestAddCmd.Flags().StringVar(&addOpts.TrustTier, "trust-tier", "T1", "trust tier: T0, T1, T2, or T3")
	treasureChestAddCmd.Flags().StringVar(&addOpts.ReviewedBy, "reviewed-by", "human", "who reviewed this source")
	treasureChestAddCmd.Flags().StringVar(&addOpts.Tags, "tags", "", "comma-separated task_type tags (default: all)")
	treasureChestAddCmd.Flags().BoolVar(&addOpts.IndexAfter, "index", false, "rebuild the compiled index immediately after adding")
	treasureChestAddCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runTreasureChestAdd(cmd, args, addOpts)
	}

	treasureChestRemoveCmd.Flags().StringVar(&removeOpts.ID, "id", "", "chest id to remove (alternative to positional path)")
	treasureChestRemoveCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runTreasureChestRemove(cmd, args, removeOpts)
	}

	treasureChestCmd.AddCommand(treasureChestAddCmd)
	treasureChestCmd.AddCommand(treasureChestRemoveCmd)
}

// --- add ---

func runTreasureChestAdd(cmd *cobra.Command, args []string, opts treasureChestAddOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	opts = treasureChestAddOptionsFromFlags(cmd, opts)

	path := args[0]
	root, err := resolveTreasureChestActionRoot(cmd, "add")
	if err != nil {
		return err
	}

	id := opts.ID
	if id == "" {
		id = treasure.DeriveChestIDFromPath(path)
	}

	if err := treasure.CheckChestIDAvailable(root, id); err != nil {
		return fmt.Errorf("treasure-chest add: %w", err)
	}

	tags := treasure.ParseTagsFlag(opts.Tags)

	indexPath, err := treasure.ExecuteAdd(root, treasure.AddOptions{
		ID:         id,
		Path:       path,
		Scope:      opts.Scope,
		TrustTier:  opts.TrustTier,
		ReviewedBy: opts.ReviewedBy,
		Tags:       tags,
	})
	if err != nil {
		return fmt.Errorf("treasure-chest add: %w", err)
	}
	refreshConfigLock(root, filepath.Join(root, "active.yaml"))

	fmt.Printf("[Strategist] add: OK (id=%s)\n", id)

	return finishChestAdd(root, indexPath, opts.IndexAfter)
}

func treasureChestAddOptionsFromFlags(cmd *cobra.Command, opts treasureChestAddOptions) treasureChestAddOptions {
	opts.ID = stringFlag(cmd, "id", opts.ID)
	opts.Scope = stringFlag(cmd, "scope", opts.Scope)
	opts.TrustTier = stringFlag(cmd, "trust-tier", opts.TrustTier)
	opts.ReviewedBy = stringFlag(cmd, "reviewed-by", opts.ReviewedBy)
	opts.Tags = stringFlag(cmd, "tags", opts.Tags)
	opts.IndexAfter = boolFlag(cmd, "index", opts.IndexAfter)
	return opts
}

// finishChestAdd reports the stale-index hint, or rebuilds the compiled index when
// --index was passed.
func finishChestAdd(root, indexPath string, indexAfter bool) error {
	if !indexAfter {
		fmt.Println("[Strategist] add: index is stale. Run: strategist treasure-chest index")
		return nil
	}
	c := compile.Compiler{}
	if err := c.CompileAll(root, indexPath); err != nil {
		return fmt.Errorf("treasure-chest add: rebuild index: %w", err)
	}
	fmt.Printf("[Strategist] add: index refreshed → %s/.compiled/\n", root)
	return nil
}

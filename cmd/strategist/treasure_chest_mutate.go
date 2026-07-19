package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/integrity"
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
	opts.ID = stringFlag(cmd, "id", opts.ID)
	opts.Scope = stringFlag(cmd, "scope", opts.Scope)
	opts.TrustTier = stringFlag(cmd, "trust-tier", opts.TrustTier)
	opts.ReviewedBy = stringFlag(cmd, "reviewed-by", opts.ReviewedBy)
	opts.Tags = stringFlag(cmd, "tags", opts.Tags)
	opts.IndexAfter = boolFlag(cmd, "index", opts.IndexAfter)

	path := args[0]
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("treasure-chest add: get cwd: %w", err)
	}
	root, _, err := resolveStrategistRoot(treasureChestRootFromCmd(cmd), cwd)
	if err != nil {
		return fmt.Errorf("treasure-chest add: %w", err)
	}

	id := opts.ID
	if id == "" {
		id = treasure.DeriveChestIDFromPath(path)
	}

	if err := treasure.CheckChestIDAvailable(root, id); err != nil {
		return fmt.Errorf("treasure-chest add: %w", err)
	}

	tags := treasure.ParseTagsFlag(opts.Tags)

	activePath := filepath.Join(root, "active.yaml")
	governedPath := filepath.Join(root, "treasure-chests.yaml")
	indexPath := filepath.Join(root, "knowledge.index.yaml")

	activeDoc, governedDoc, indexDoc, err := treasure.LoadChestYAMLDocs(activePath, governedPath, indexPath)
	if err != nil {
		return fmt.Errorf("treasure-chest add: %w", err)
	}

	if err := treasure.ApplyAddMutations(activeDoc, governedDoc, indexDoc, id, path, opts.Scope, opts.TrustTier, opts.ReviewedBy, tags); err != nil {
		return fmt.Errorf("treasure-chest add: %w", err)
	}

	written, err := treasure.WriteYAMLNodes(
		treasure.YAMLWrite{Path: activePath, Doc: activeDoc},
		treasure.YAMLWrite{Path: governedPath, Doc: governedDoc},
		treasure.YAMLWrite{Path: indexPath, Doc: indexDoc},
	)
	if err != nil {
		return fmt.Errorf("treasure-chest add: partial write after %v: %w", written, err)
	}
	refreshConfigLock(root, activePath)

	fmt.Printf("[Strategist] add: OK (id=%s)\n", id)

	return finishChestAdd(root, indexPath, opts.IndexAfter)
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

// refreshConfigLock re-seals the config integrity lock after a CLI command
// legitimately writes active.yaml. Without this, the next command's
// integrity.IsModified check sees the mtime change and falsely warns that
// active.yaml was "modified outside the CLI" — even though this command is
// the CLI. Best-effort: a failure here only means the next run may show a
// stale-lock warning, not a functional break.
func refreshConfigLock(root, activePath string) {
	lockPath := filepath.Join(root, ".config.lock")
	if err := integrity.WriteLock(activePath, lockPath); err != nil {
		fmt.Fprintf(os.Stderr, "[Strategist] WARN: could not refresh config lock: %v\n", err)
	}
}

// --- remove ---

func runTreasureChestRemove(cmd *cobra.Command, args []string, opts treasureChestRemoveOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	opts.ID = stringFlag(cmd, "id", opts.ID)

	var pathArg string
	if len(args) == 1 {
		pathArg = args[0]
	}
	if pathArg == "" && opts.ID == "" {
		return fmt.Errorf("treasure-chest remove: provide a path or --id")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("treasure-chest remove: get cwd: %w", err)
	}
	root, _, err := resolveStrategistRoot(treasureChestRootFromCmd(cmd), cwd)
	if err != nil {
		return fmt.Errorf("treasure-chest remove: %w", err)
	}

	id, err := treasure.ResolveRemoveTarget(root, pathArg, opts.ID)
	if err != nil {
		return fmt.Errorf("treasure-chest remove: %w", err)
	}

	paths := treasure.NewChestPaths(root)
	docs, err := treasure.LoadRemoveDocs(paths)
	if err != nil {
		return fmt.Errorf("treasure-chest remove: %w", err)
	}

	if err := treasure.ApplyRemoveMutations(docs, id); err != nil {
		return fmt.Errorf("treasure-chest remove: %w", err)
	}

	written, err := treasure.WriteRemoveDocs(paths, docs)
	if err != nil {
		return fmt.Errorf("treasure-chest remove: partial write after %v: %w", written, err)
	}
	refreshConfigLock(root, paths.Active)

	reportRemoveResult(id, len(docs.Jewels) > 0)
	return nil
}

func reportRemoveResult(id string, hasJewels bool) {
	fmt.Printf("[Strategist] remove: OK (id=%s)\n", id)
	if hasJewels {
		fmt.Printf("[Strategist] remove: %q's jewels marked deprecated in jewels.yaml\n", id)
	}
	fmt.Println("[Strategist] remove: index is stale. Run: strategist treasure-chest index")
}

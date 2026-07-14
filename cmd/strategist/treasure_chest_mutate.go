package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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

var (
	treasureChestAddID         string
	treasureChestAddScope      string
	treasureChestAddTrustTier  string
	treasureChestAddReviewedBy string
	treasureChestAddTags       string
	treasureChestAddIndexAfter bool
	treasureChestRemoveID      string
)

var treasureChestAddCmd = &cobra.Command{
	Use:   "add <path>",
	Short: "Register a new treasure chest across active/governed/indexed layers",
	Long: `Add a treasure chest source, updating active.yaml, treasure-chests.yaml, and
knowledge.index.yaml together in one pass. Existing file comments/formatting are preserved.

Leaves the compiled index (.compiled/.index.gz) stale unless --index is also passed.`,
	Args: cobra.ExactArgs(1),
	RunE: runTreasureChestAdd,
}

var treasureChestRemoveCmd = &cobra.Command{
	Use:   "remove [path]",
	Short: "Tombstone a treasure chest (mark inactive, do not hard-delete)",
	Long: `Remove a treasure chest from active.yaml's active declarations and mark it
status: inactive in treasure-chests.yaml and knowledge.index.yaml, preserving an audit
trail instead of hard-deleting the entry. Resolve by positional path or --id; if both are
given and disagree, the command stops with an ambiguity error.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTreasureChestRemove,
}

func init() {
	treasureChestAddCmd.Flags().StringVar(&treasureChestAddID, "id", "", "chest id (default: derived from the last path segment)")
	treasureChestAddCmd.Flags().StringVar(&treasureChestAddScope, "scope", "all", "slot scope: all, discovery, refinement, or execution")
	treasureChestAddCmd.Flags().StringVar(&treasureChestAddTrustTier, "trust-tier", "T1", "trust tier: T0, T1, T2, or T3")
	treasureChestAddCmd.Flags().StringVar(&treasureChestAddReviewedBy, "reviewed-by", "human", "who reviewed this source")
	treasureChestAddCmd.Flags().StringVar(&treasureChestAddTags, "tags", "", "comma-separated task_type tags (default: all)")
	treasureChestAddCmd.Flags().BoolVar(&treasureChestAddIndexAfter, "index", false, "rebuild the compiled index immediately after adding")

	treasureChestRemoveCmd.Flags().StringVar(&treasureChestRemoveID, "id", "", "chest id to remove (alternative to positional path)")

	treasureChestCmd.AddCommand(treasureChestAddCmd)
	treasureChestCmd.AddCommand(treasureChestRemoveCmd)
}

// --- add ---

func runTreasureChestAdd(cmd *cobra.Command, args []string) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}

	path := args[0]
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("treasure-chest add: get cwd: %w", err)
	}
	root, _, err := resolveStrategistRoot(treasureChestRoot, cwd)
	if err != nil {
		return fmt.Errorf("treasure-chest add: %w", err)
	}

	id := treasureChestAddID
	if id == "" {
		id = deriveChestIDFromPath(path)
	}

	if err := checkChestIDAvailable(root, id); err != nil {
		return err
	}

	tags := parseTagsFlag(treasureChestAddTags)

	activePath := filepath.Join(root, "active.yaml")
	governedPath := filepath.Join(root, "treasure-chests.yaml")
	indexPath := filepath.Join(root, "knowledge.index.yaml")

	activeDoc, governedDoc, indexDoc, err := loadChestYAMLDocs("treasure-chest add", activePath, governedPath, indexPath)
	if err != nil {
		return err
	}

	if err := applyAddMutations(activeDoc, governedDoc, indexDoc, id, path, treasureChestAddScope, treasureChestAddTrustTier, treasureChestAddReviewedBy, tags); err != nil {
		return err
	}

	written, err := writeYAMLNodes(
		yamlWrite{activePath, activeDoc},
		yamlWrite{governedPath, governedDoc},
		yamlWrite{indexPath, indexDoc},
	)
	if err != nil {
		return fmt.Errorf("treasure-chest add: partial write after %v: %w", written, err)
	}

	fmt.Printf("[Strategist] treasure-chest add: registered %q (id=%s) in active.yaml, treasure-chests.yaml, knowledge.index.yaml\n", path, id)

	return finishChestAdd(root, indexPath)
}

// loadChestYAMLDocs reads the three treasure-chest YAML documents shared by add/remove,
// wrapping any read error with opPrefix.
func loadChestYAMLDocs(opPrefix, activePath, governedPath, indexPath string) (activeDoc, governedDoc, indexDoc *yaml.Node, err error) {
	activeDoc, err = readYAMLNode(activePath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%s: %w", opPrefix, err)
	}
	governedDoc, err = readYAMLNode(governedPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%s: %w", opPrefix, err)
	}
	indexDoc, err = readYAMLNode(indexPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%s: %w", opPrefix, err)
	}
	return activeDoc, governedDoc, indexDoc, nil
}

// applyAddMutations appends the new chest entry to each of the three loaded documents.
func applyAddMutations(activeDoc, governedDoc, indexDoc *yaml.Node, id, path, scope, trustTier, reviewedBy string, tags []string) error {
	if err := appendActiveChestEntry(activeDoc, id, path, scope); err != nil {
		return fmt.Errorf("treasure-chest add: %w", err)
	}
	if err := appendGovernedChestEntry(governedDoc, id, path, trustTier, reviewedBy, tags); err != nil {
		return fmt.Errorf("treasure-chest add: %w", err)
	}
	if err := appendIndexedSourceEntry(indexDoc, id, path, tags); err != nil {
		return fmt.Errorf("treasure-chest add: %w", err)
	}
	return nil
}

// finishChestAdd reports the stale-index hint, or rebuilds the compiled index when
// --index was passed.
func finishChestAdd(root, indexPath string) error {
	if !treasureChestAddIndexAfter {
		fmt.Println("[Strategist] treasure-chest add: compiled index is now stale. Run: strategist treasure-chest --index")
		return nil
	}
	c := compile.Compiler{}
	if err := c.CompileAll(root, indexPath); err != nil {
		return fmt.Errorf("treasure-chest add: rebuild index: %w", err)
	}
	fmt.Printf("[Strategist] treasure-chest add: compiled index refreshed → %s/.compiled/\n", root)
	return nil
}

// --- remove ---

func runTreasureChestRemove(cmd *cobra.Command, args []string) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}

	var pathArg string
	if len(args) == 1 {
		pathArg = args[0]
	}
	if pathArg == "" && treasureChestRemoveID == "" {
		return fmt.Errorf("treasure-chest remove: provide a path or --id")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("treasure-chest remove: get cwd: %w", err)
	}
	root, _, err := resolveStrategistRoot(treasureChestRoot, cwd)
	if err != nil {
		return fmt.Errorf("treasure-chest remove: %w", err)
	}

	id, err := resolveRemoveTarget(root, pathArg, treasureChestRemoveID)
	if err != nil {
		return err
	}

	paths := newChestPaths(root)
	docs, err := loadRemoveDocs(paths)
	if err != nil {
		return err
	}

	if err := applyRemoveMutations(docs, id); err != nil {
		return err
	}

	written, err := writeRemoveDocs(paths, docs)
	if err != nil {
		return fmt.Errorf("treasure-chest remove: partial write after %v: %w", written, err)
	}

	reportRemoveResult(id, docs.hasJewels)
	return nil
}

// chestPaths bundles the treasure-chest artifact paths derived from root.
type chestPaths struct {
	active, governed, index, jewels string
}

func newChestPaths(root string) chestPaths {
	return chestPaths{
		active:   filepath.Join(root, "active.yaml"),
		governed: filepath.Join(root, "treasure-chests.yaml"),
		index:    filepath.Join(root, "knowledge.index.yaml"),
		jewels:   filepath.Join(root, "jewels.yaml"),
	}
}

// chestDocSet holds the parsed YAML documents touched by a remove/tombstone operation.
// jewels.yaml is optional — a chest may have no jewels yet, so hasJewels tracks whether
// it was found and should be written back.
type chestDocSet struct {
	active, governed, index, jewels *yaml.Node
	hasJewels                       bool
}

func loadRemoveDocs(p chestPaths) (chestDocSet, error) {
	var docs chestDocSet
	var err error
	docs.active, err = readYAMLNode(p.active)
	if err != nil {
		return docs, fmt.Errorf("treasure-chest remove: %w", err)
	}
	docs.governed, err = readYAMLNode(p.governed)
	if err != nil {
		return docs, fmt.Errorf("treasure-chest remove: %w", err)
	}
	docs.index, err = readYAMLNode(p.index)
	if err != nil {
		return docs, fmt.Errorf("treasure-chest remove: %w", err)
	}
	if jewelsDoc, jewelsErr := readYAMLNode(p.jewels); jewelsErr == nil {
		docs.jewels = jewelsDoc
		docs.hasJewels = true
	}
	return docs, nil
}

func applyRemoveMutations(docs chestDocSet, id string) error {
	if err := removeActiveChestEntry(docs.active, id); err != nil {
		return fmt.Errorf("treasure-chest remove: %w", err)
	}
	if err := markGovernedChestInactive(docs.governed, id); err != nil {
		return fmt.Errorf("treasure-chest remove: %w", err)
	}
	if err := markIndexedSourceInactive(docs.index, id); err != nil {
		return fmt.Errorf("treasure-chest remove: %w", err)
	}
	if docs.hasJewels {
		if err := markJewelsDeprecatedForChest(docs.jewels, id); err != nil {
			return fmt.Errorf("treasure-chest remove: %w", err)
		}
	}
	return nil
}

func writeRemoveDocs(p chestPaths, docs chestDocSet) ([]string, error) {
	writes := []yamlWrite{
		{p.active, docs.active},
		{p.governed, docs.governed},
		{p.index, docs.index},
	}
	if docs.hasJewels {
		writes = append(writes, yamlWrite{p.jewels, docs.jewels})
	}
	return writeYAMLNodes(writes...)
}

func reportRemoveResult(id string, hasJewels bool) {
	fmt.Printf("[Strategist] treasure-chest remove: %q removed from active.yaml, marked inactive in treasure-chests.yaml and knowledge.index.yaml\n", id)
	if hasJewels {
		fmt.Printf("[Strategist] treasure-chest remove: %q's jewels marked deprecated in jewels.yaml\n", id)
	}
	fmt.Println("[Strategist] treasure-chest remove: compiled index is now stale. Run: strategist treasure-chest --index")
}

// resolveRemoveTarget resolves the chest id to remove from a positional path and/or
// --id flag, rejecting ambiguous input (both given but disagreeing).
func resolveRemoveTarget(root, pathArg, idFlag string) (string, error) {
	if pathArg == "" {
		return idFlag, nil
	}

	activeChests, err := loadActiveChests(root)
	if err != nil {
		return "", fmt.Errorf("treasure-chest remove: %w", err)
	}
	var matches []string
	for _, ac := range activeChests {
		if ac.Path == pathArg {
			matches = append(matches, ac.ID)
		}
	}
	switch len(matches) {
	case 0:
		if idFlag != "" {
			return idFlag, nil
		}
		return "", fmt.Errorf("treasure-chest remove: no chest registered with path %q", pathArg)
	case 1:
		if idFlag != "" && idFlag != matches[0] {
			return "", fmt.Errorf("treasure-chest remove: ambiguous — path %q resolves to id %q but --id=%q was given", pathArg, matches[0], idFlag)
		}
		return matches[0], nil
	default:
		return "", fmt.Errorf("treasure-chest remove: ambiguous — multiple chests share path %q (%s); use --id", pathArg, strings.Join(matches, ", "))
	}
}

// checkChestIDAvailable errors if id is already registered and active.
func checkChestIDAvailable(root, id string) error {
	activeChests, err := loadActiveChests(root)
	if err != nil {
		return fmt.Errorf("treasure-chest add: %w", err)
	}
	for _, ac := range activeChests {
		if ac.ID == id {
			return fmt.Errorf("treasure-chest add: id %q is already registered in active.yaml; use a different --id or remove it first", id)
		}
	}
	return nil
}

func deriveChestIDFromPath(path string) string {
	path = strings.TrimRight(path, "/")
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

func parseTagsFlag(tags string) []string {
	if strings.TrimSpace(tags) == "" {
		return []string{"all"}
	}
	parts := strings.Split(tags, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"all"}
	}
	return out
}

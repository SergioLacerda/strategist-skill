package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreasureChestRemove_ByID(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestRemoveCmd, "id", "source")

	require.NoError(t, treasureChestRemoveCmd.RunE(treasureChestRemoveCmd, nil))

	active, err := treasure.LoadActiveChests(dir)
	require.NoError(t, err)
	assert.Empty(t, active)

	governed, err := treasure.LoadGoverned(dir)
	require.NoError(t, err)
	require.Contains(t, governed, "source")
}

func TestTreasureChestRemove_ByPath(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	setTreasureChestRoot(t, dir)

	require.NoError(t, treasureChestRemoveCmd.RunE(treasureChestRemoveCmd, []string{".sdd/source"}))

	active, err := treasure.LoadActiveChests(dir)
	require.NoError(t, err)
	assert.Empty(t, active)
}

func TestTreasureChestRemove_CascadesJewelDeprecation(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "jewels.yaml"), []byte(`
jewels:
  - id: jewel-1
    chest_id: source
    statement: "A useful fact."
    source_refs: ["source#x"]
    trust: T1
    status: accepted
    reviewed_by: agent
`), 0o644))
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestRemoveCmd, "id", "source")

	require.NoError(t, treasureChestRemoveCmd.RunE(treasureChestRemoveCmd, nil))

	raw, err := os.ReadFile(filepath.Join(dir, "jewels.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "status: deprecated")
	assert.Contains(t, string(raw), "history:")
	assert.Contains(t, string(raw), "id: jewel-1") // entry still present, not deleted
}

func TestTreasureChestRemove_NoJewelsFileIsNoop(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestRemoveCmd, "id", "source")

	// No jewels.yaml present in minimalTreasureChestRoot — must not error.
	require.NoError(t, treasureChestRemoveCmd.RunE(treasureChestRemoveCmd, nil))
}

func TestTreasureChestRemove_TombstonesInsteadOfDeleting(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestRemoveCmd, "id", "source")

	require.NoError(t, treasureChestRemoveCmd.RunE(treasureChestRemoveCmd, nil))

	raw, err := os.ReadFile(filepath.Join(dir, "treasure-chests.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "status: inactive")
	assert.Contains(t, string(raw), "id: source") // entry still present, not hard-deleted

	rawIdx, err := os.ReadFile(filepath.Join(dir, "knowledge.index.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(rawIdx), "status: inactive")
}

func TestTreasureChestRemove_NoPathOrIDErrors(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	setTreasureChestRoot(t, dir)

	err := treasureChestRemoveCmd.RunE(treasureChestRemoveCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provide a path or --id")
}

func TestTreasureChestRemove_PathIDMismatchIsAmbiguous(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestRemoveCmd, "id", "does-not-match")

	err := treasureChestRemoveCmd.RunE(treasureChestRemoveCmd, []string{".sdd/source"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous")
}

// --- runTreasureChestRemove: resolveStrategistRoot error ---

func TestTreasureChestRemove_ResolveRootError(t *testing.T) {
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	setTreasureChestRoot(t, "")
	setCmdFlag(t, treasureChestRemoveCmd, "id", "source")

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	require.NoError(t, os.Chdir(t.TempDir()))

	err = treasureChestRemoveCmd.RunE(treasureChestRemoveCmd, nil)
	require.Error(t, err)
	assert.NotEmpty(t, err.Error())
}

// --- runTreasureChestRemove: treasure.LoadRemoveDocs / treasure.ApplyRemoveMutations / writeRemoveDocs error at command level ---

func TestTreasureChestRemove_MissingGovernedFileErrors(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	require.NoError(t, os.Remove(filepath.Join(dir, "treasure-chests.yaml")))
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestRemoveCmd, "id", "source")

	err := treasureChestRemoveCmd.RunE(treasureChestRemoveCmd, nil)
	require.Error(t, err)
	assert.NotEmpty(t, err.Error())
}

func TestTreasureChestRemove_ApplyMutationsErrorAtCommandLevel(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	// active.yaml has no treasure_chests key at all -> removeActiveChestEntry fails
	// inside treasure.ApplyRemoveMutations, reached via --id (skips resolveRemoveTarget's path lookup).
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: epic
base_path: .analysis
slots:
  discovery: brainstorming
  refinement: openspec-explore
  execution: sdd-ask
`), 0o644))
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestRemoveCmd, "id", "source")

	err := treasureChestRemoveCmd.RunE(treasureChestRemoveCmd, nil)
	require.Error(t, err)
	assert.NotEmpty(t, err.Error())
	assert.Contains(t, err.Error(), "no treasure_chests declared")
}

func TestTreasureChestRemove_WriteErrorAtCommandLevel(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := minimalTreasureChestRoot(t)
	// WriteYAMLNodes writes each governance file through writeFileAtomic
	// (temp file + os.Rename, see internal/treasure/yaml_node.go, mission
	// 2026-07-22-yaml-write-atomicity). os.Rename does not check the
	// destination file's own permission bits, only that its parent
	// directory is writable — so chmod'ing treasure-chests.yaml itself no
	// longer forces a write failure. Chmod the directory read-only instead:
	// reads (LoadRemoveDocs) only need r-x and still succeed, but
	// os.CreateTemp for the first write in the batch fails, and the command
	// must still surface that as an error.
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestRemoveCmd, "id", "source")

	err := treasureChestRemoveCmd.RunE(treasureChestRemoveCmd, nil)
	require.Error(t, err)
	assert.NotEmpty(t, err.Error())
}

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetTreasureChestMutateFlags(t *testing.T) {
	t.Helper()
	origID := cmdFlagString(t, treasureChestAddCmd, "id")
	origScope := cmdFlagString(t, treasureChestAddCmd, "scope")
	origTrustTier := cmdFlagString(t, treasureChestAddCmd, "trust-tier")
	origReviewedBy := cmdFlagString(t, treasureChestAddCmd, "reviewed-by")
	origTags := cmdFlagString(t, treasureChestAddCmd, "tags")
	origIndexAfter := cmdFlagBool(t, treasureChestAddCmd, "index")
	origRemoveID := cmdFlagString(t, treasureChestRemoveCmd, "id")
	t.Cleanup(func() {
		setCmdFlag(t, treasureChestAddCmd, "id", origID)
		setCmdFlag(t, treasureChestAddCmd, "scope", origScope)
		setCmdFlag(t, treasureChestAddCmd, "trust-tier", origTrustTier)
		setCmdFlag(t, treasureChestAddCmd, "reviewed-by", origReviewedBy)
		setCmdFlag(t, treasureChestAddCmd, "tags", origTags)
		setCmdFlag(t, treasureChestAddCmd, "index", fmt.Sprint(origIndexAfter))
		setCmdFlag(t, treasureChestRemoveCmd, "id", origRemoveID)
	})
	setCmdFlag(t, treasureChestAddCmd, "id", "")
	setCmdFlag(t, treasureChestAddCmd, "scope", "all")
	setCmdFlag(t, treasureChestAddCmd, "trust-tier", "T1")
	setCmdFlag(t, treasureChestAddCmd, "reviewed-by", "human")
	setCmdFlag(t, treasureChestAddCmd, "tags", "")
	setCmdFlag(t, treasureChestAddCmd, "index", "false")
	setCmdFlag(t, treasureChestRemoveCmd, "id", "")
}

func TestTreasureChestAdd_DefaultsAllThreeFiles(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	setTreasureChestRoot(t, dir)

	require.NoError(t, treasureChestAddCmd.RunE(treasureChestAddCmd, []string{"/tmp/new-chest"}))

	active, err := treasure.LoadActiveChests(dir)
	require.NoError(t, err)
	require.Len(t, active, 2)
	assert.Equal(t, "new-chest", active[1].ID)
	assert.Equal(t, "/tmp/new-chest", active[1].Path)
	assert.Equal(t, []string{"all"}, []string(active[1].Scope))

	governed, err := treasure.LoadGoverned(dir)
	require.NoError(t, err)
	require.Contains(t, governed, "new-chest")
	assert.Equal(t, "T1", governed["new-chest"].Trust.Tier)
	assert.Equal(t, "human", governed["new-chest"].Trust.ReviewedBy)

	indexed, err := treasure.LoadIndexed(dir)
	require.NoError(t, err)
	assert.True(t, indexed["new-chest"])
}

func TestTreasureChestAdd_ExplicitFlagsOverrideDefaults(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestAddCmd, "id", "custom-id")
	setCmdFlag(t, treasureChestAddCmd, "scope", "discovery")
	setCmdFlag(t, treasureChestAddCmd, "trust-tier", "T0")
	setCmdFlag(t, treasureChestAddCmd, "reviewed-by", "auto")
	setCmdFlag(t, treasureChestAddCmd, "tags", "foo, bar")

	require.NoError(t, treasureChestAddCmd.RunE(treasureChestAddCmd, []string{"/tmp/new-chest"}))

	active, err := treasure.LoadActiveChests(dir)
	require.NoError(t, err)
	require.Len(t, active, 2)
	assert.Equal(t, "custom-id", active[1].ID)
	assert.Equal(t, []string{"discovery"}, []string(active[1].Scope))

	governed, err := treasure.LoadGoverned(dir)
	require.NoError(t, err)
	require.Contains(t, governed, "custom-id")
	assert.Equal(t, "T0", governed["custom-id"].Trust.Tier)
	assert.Equal(t, "auto", governed["custom-id"].Trust.ReviewedBy)
}

func TestTreasureChestAdd_PreservesExistingComments(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	setTreasureChestRoot(t, dir)

	tcPath := filepath.Join(dir, "treasure-chests.yaml")
	orig, err := os.ReadFile(tcPath)
	require.NoError(t, err)
	withComment := "# custom marker comment — must survive add\n" + string(orig)
	require.NoError(t, os.WriteFile(tcPath, []byte(withComment), 0o644))

	require.NoError(t, treasureChestAddCmd.RunE(treasureChestAddCmd, []string{"/tmp/new-chest"}))

	raw, err := os.ReadFile(tcPath)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "custom marker comment — must survive add")
}

func TestTreasureChestAdd_DuplicateIDErrors(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestAddCmd, "id", "source") // already registered by minimalTreasureChestRoot

	err := treasureChestAddCmd.RunE(treasureChestAddCmd, []string{"/tmp/new-chest"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

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

func TestDeriveChestIDFromPath(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "docs", treasure.DeriveChestIDFromPath("/abs/path/to/docs"))
	assert.Equal(t, "docs", treasure.DeriveChestIDFromPath("/abs/path/to/docs/"))
	assert.Equal(t, "relative", treasure.DeriveChestIDFromPath("relative"))
}

func TestParseTagsFlag(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"all"}, treasure.ParseTagsFlag(""))
	assert.Equal(t, []string{"foo", "bar"}, treasure.ParseTagsFlag("foo, bar"))
}

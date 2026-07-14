package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetTreasureChestMutateFlags(t *testing.T) {
	t.Helper()
	origID := treasureChestAddID
	origScope := treasureChestAddScope
	origTrustTier := treasureChestAddTrustTier
	origReviewedBy := treasureChestAddReviewedBy
	origTags := treasureChestAddTags
	origIndexAfter := treasureChestAddIndexAfter
	origRemoveID := treasureChestRemoveID
	t.Cleanup(func() {
		treasureChestAddID = origID
		treasureChestAddScope = origScope
		treasureChestAddTrustTier = origTrustTier
		treasureChestAddReviewedBy = origReviewedBy
		treasureChestAddTags = origTags
		treasureChestAddIndexAfter = origIndexAfter
		treasureChestRemoveID = origRemoveID
	})
	treasureChestAddID = ""
	treasureChestAddScope = "all"
	treasureChestAddTrustTier = "T1"
	treasureChestAddReviewedBy = "human"
	treasureChestAddTags = ""
	treasureChestAddIndexAfter = false
	treasureChestRemoveID = ""
}

func TestTreasureChestAdd_DefaultsAllThreeFiles(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	treasureChestRoot = dir

	require.NoError(t, treasureChestAddCmd.RunE(treasureChestAddCmd, []string{"/tmp/new-chest"}))

	active, err := loadActiveChests(dir)
	require.NoError(t, err)
	require.Len(t, active, 2)
	assert.Equal(t, "new-chest", active[1].ID)
	assert.Equal(t, "/tmp/new-chest", active[1].Path)
	assert.Equal(t, []string{"all"}, []string(active[1].Scope))

	governed, err := loadGoverned(dir)
	require.NoError(t, err)
	require.Contains(t, governed, "new-chest")
	assert.Equal(t, "T1", governed["new-chest"].Trust.Tier)
	assert.Equal(t, "human", governed["new-chest"].Trust.ReviewedBy)

	indexed, err := loadIndexed(dir)
	require.NoError(t, err)
	assert.True(t, indexed["new-chest"])
}

func TestTreasureChestAdd_ExplicitFlagsOverrideDefaults(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	treasureChestRoot = dir
	treasureChestAddID = "custom-id"
	treasureChestAddScope = "discovery"
	treasureChestAddTrustTier = "T0"
	treasureChestAddReviewedBy = "auto"
	treasureChestAddTags = "foo, bar"

	require.NoError(t, treasureChestAddCmd.RunE(treasureChestAddCmd, []string{"/tmp/new-chest"}))

	active, err := loadActiveChests(dir)
	require.NoError(t, err)
	require.Len(t, active, 2)
	assert.Equal(t, "custom-id", active[1].ID)
	assert.Equal(t, []string{"discovery"}, []string(active[1].Scope))

	governed, err := loadGoverned(dir)
	require.NoError(t, err)
	require.Contains(t, governed, "custom-id")
	assert.Equal(t, "T0", governed["custom-id"].Trust.Tier)
	assert.Equal(t, "auto", governed["custom-id"].Trust.ReviewedBy)
}

func TestTreasureChestAdd_PreservesExistingComments(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	treasureChestRoot = dir

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
	treasureChestRoot = dir
	treasureChestAddID = "source" // already registered by minimalTreasureChestRoot

	err := treasureChestAddCmd.RunE(treasureChestAddCmd, []string{"/tmp/new-chest"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestTreasureChestRemove_ByID(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	treasureChestRoot = dir
	treasureChestRemoveID = "source"

	require.NoError(t, treasureChestRemoveCmd.RunE(treasureChestRemoveCmd, nil))

	active, err := loadActiveChests(dir)
	require.NoError(t, err)
	assert.Empty(t, active)

	governed, err := loadGoverned(dir)
	require.NoError(t, err)
	require.Contains(t, governed, "source")
}

func TestTreasureChestRemove_ByPath(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	treasureChestRoot = dir

	require.NoError(t, treasureChestRemoveCmd.RunE(treasureChestRemoveCmd, []string{".sdd/source"}))

	active, err := loadActiveChests(dir)
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
    status: active
    reviewed_by: agent
`), 0o644))
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	treasureChestRoot = dir
	treasureChestRemoveID = "source"

	require.NoError(t, treasureChestRemoveCmd.RunE(treasureChestRemoveCmd, nil))

	raw, err := os.ReadFile(filepath.Join(dir, "jewels.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "status: deprecated")
	assert.Contains(t, string(raw), "id: jewel-1") // entry still present, not deleted
}

func TestTreasureChestRemove_NoJewelsFileIsNoop(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	treasureChestRoot = dir
	treasureChestRemoveID = "source"

	// No jewels.yaml present in minimalTreasureChestRoot — must not error.
	require.NoError(t, treasureChestRemoveCmd.RunE(treasureChestRemoveCmd, nil))
}

func TestTreasureChestRemove_TombstonesInsteadOfDeleting(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	treasureChestRoot = dir
	treasureChestRemoveID = "source"

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
	treasureChestRoot = dir

	err := treasureChestRemoveCmd.RunE(treasureChestRemoveCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provide a path or --id")
}

func TestTreasureChestRemove_PathIDMismatchIsAmbiguous(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	treasureChestRoot = dir
	treasureChestRemoveID = "does-not-match"

	err := treasureChestRemoveCmd.RunE(treasureChestRemoveCmd, []string{".sdd/source"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous")
}

func TestDeriveChestIDFromPath(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "docs", deriveChestIDFromPath("/abs/path/to/docs"))
	assert.Equal(t, "docs", deriveChestIDFromPath("/abs/path/to/docs/"))
	assert.Equal(t, "relative", deriveChestIDFromPath("relative"))
}

func TestParseTagsFlag(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"all"}, parseTagsFlag(""))
	assert.Equal(t, []string{"foo", "bar"}, parseTagsFlag("foo, bar"))
}

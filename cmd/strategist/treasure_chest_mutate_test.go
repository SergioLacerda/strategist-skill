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

// --- runTreasureChestAdd: resolveStrategistRoot error ---

func TestTreasureChestAdd_ResolveRootError(t *testing.T) {
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	setTreasureChestRoot(t, "") // forces findStrategistRoot(cwd)

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	require.NoError(t, os.Chdir(t.TempDir()))

	err = treasureChestAddCmd.RunE(treasureChestAddCmd, []string{"/tmp/new-chest"})
	require.Error(t, err)
	assert.NotEmpty(t, err.Error())
}

// --- runTreasureChestAdd: loadChestYAMLDocs / writeYAMLNodes error at command level ---

func TestTreasureChestAdd_MissingGovernedFileErrors(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	require.NoError(t, os.Remove(filepath.Join(dir, "treasure-chests.yaml")))
	resetTreasureChestFlags(t)
	resetTreasureChestMutateFlags(t)
	setTreasureChestRoot(t, dir)

	err := treasureChestAddCmd.RunE(treasureChestAddCmd, []string{"/tmp/new-chest"})
	require.Error(t, err)
	assert.NotEmpty(t, err.Error())
}

// --- finishChestAdd ---

func TestFinishChestAdd_IndexAfterSuccess(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	setCmdFlag(t, treasureChestAddCmd, "index", "true")

	indexPath := filepath.Join(dir, "knowledge.index.yaml")
	out := captureStdout(t, func() {
		require.NoError(t, finishChestAdd(dir, indexPath, true))
	})
	assert.Contains(t, out, "index refreshed")

	_, err := os.Stat(filepath.Join(dir, ".compiled"))
	require.NoError(t, err)
}

func TestFinishChestAdd_IndexAfterCompileError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := minimalTreasureChestRoot(t)
	setCmdFlag(t, treasureChestAddCmd, "index", "true")

	compiledDir := filepath.Join(dir, ".compiled")
	require.NoError(t, os.MkdirAll(compiledDir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(compiledDir, 0o755) })

	indexPath := filepath.Join(dir, "knowledge.index.yaml")
	err := finishChestAdd(dir, indexPath, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rebuild index")
}

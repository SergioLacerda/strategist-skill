package treasurecli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTreasureChestItemsShow_JewelTable and its JSON sibling below cover the
// one gap this file's coverage push actually needs: no existing test ever
// calls showJewelItem with a *valid* format — TestTreasureChestItemsCommands_UnknownFormat
// (coverage_more_test.go) only exercises jewel-1 with an invalid format, and
// TestTreasureChestRenderers_ClosedStdoutErrors calls renderJewelShowTable/JSON
// directly, bypassing showJewelItem's own switch entirely. The potion side
// already has happy-path coverage (TestTreasureChestItemsShow_Potion/
// PotionJSONFormat in treasure_chest_potions_test.go) — this file does not
// duplicate that.

func TestTreasureChestItemsShow_JewelTable(t *testing.T) {
	dir := itemsTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)
	attachMissionRun(t, treasureChestItemsShowCmd)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestItemsShowCmd.RunE(treasureChestItemsShowCmd, []string{"jewel-1"}))
	})
	assert.Contains(t, out, "jewel-1")
	assert.Contains(t, out, "Widgets require explicit teardown.")
}

func TestTreasureChestItemsShow_JewelJSONFormat(t *testing.T) {
	dir := itemsTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestItemsShowCmd, "format", "json")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestItemsShowCmd.RunE(treasureChestItemsShowCmd, []string{"jewel-1"}))
	})
	var decoded jsonJewelShowEntry
	require.NoError(t, json.Unmarshal([]byte(out), &decoded))
	assert.Equal(t, "jewel-1", decoded.ID)
}

// TestTreasureChestItemsShow_PotionUnknownFormat covers showPotionItem's
// default (invalid-format) case — the mirror-image gap of the jewel side:
// the existing potion tests only ever use valid formats.
func TestTreasureChestItemsShow_PotionUnknownFormat(t *testing.T) {
	dir := itemsTestRoot(t, oneProposedJewelYAML)
	writePotionsFile(t, dir, oneProposedPotionYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestItemsShowCmd, "format", "xml")

	err := treasureChestItemsShowCmd.RunE(treasureChestItemsShowCmd, []string{"potion-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown --format")
}

// TestTreasureChestItemsShow_JewelLoadErrorPropagates and its potion
// counterpart mirror TestTreasureChestItemsList_JewelLoadErrorPropagates /
// _PotionLoadErrorPropagates (treasure_chest_items_list_test.go) — the same
// malformed-YAML technique, applied to the `show` subcommand instead of
// `list`.

func TestTreasureChestItemsShow_JewelLoadErrorPropagates(t *testing.T) {
	dir := itemsTestRoot(t, oneProposedJewelYAML)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "jewels.yaml"), []byte("jewels: [unterminated\n"), 0o644))
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)

	err := treasureChestItemsShowCmd.RunE(treasureChestItemsShowCmd, []string{"jewel-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest items show")
}

func TestTreasureChestItemsShow_PotionLoadErrorPropagates(t *testing.T) {
	dir := itemsTestRoot(t, oneProposedJewelYAML)
	writePotionsFile(t, dir, "potions: [unterminated\n")
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)

	// "potion-1" doesn't match the fixture jewel ("jewel-1"), so
	// runTreasureChestItemsShow falls through to loading potions, where the
	// malformed potions.yaml triggers the error.
	err := treasureChestItemsShowCmd.RunE(treasureChestItemsShowCmd, []string{"potion-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest items show")
}

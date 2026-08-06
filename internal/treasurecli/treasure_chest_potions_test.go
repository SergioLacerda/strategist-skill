package treasurecli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const oneProposedPotionYAML = `
schema_version: "1"
potions:
  - id: potion-1
    chest_id: runbooks
    runbook_ref: docs/runbooks/sample.md
    when_to_use: "When sample breaks."
    trust: T2
    status: proposed
    source_refs: ["docs/runbooks/sample.md"]
    reviewed_by: agent
`

func writePotionsFile(t *testing.T, dir, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "potions.yaml"), []byte(content), 0o644))
}

func readPotionsFile(t *testing.T, dir string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "potions.yaml"))
	require.NoError(t, err)
	return string(raw)
}

func TestTreasureChestItemsShow_Potion(t *testing.T) {
	dir := itemsTestRoot(t, oneProposedJewelYAML)
	writePotionsFile(t, dir, oneProposedPotionYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestItemsShowCmd.RunE(treasureChestItemsShowCmd, []string{"potion-1"}))
	})
	assert.Contains(t, out, "potion-1")
	assert.Contains(t, out, "docs/runbooks/sample.md")
	assert.Contains(t, out, "When sample breaks.")
}

func TestTreasureChestItemsShow_PotionJSONFormat(t *testing.T) {
	dir := itemsTestRoot(t, oneProposedJewelYAML)
	writePotionsFile(t, dir, oneProposedPotionYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestItemsShowCmd, "format", "json")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestItemsShowCmd.RunE(treasureChestItemsShowCmd, []string{"potion-1"}))
	})
	var decoded jsonPotionShowEntry
	require.NoError(t, json.Unmarshal([]byte(out), &decoded))
	assert.Equal(t, "potion-1", decoded.ID)
	assert.Equal(t, "docs/runbooks/sample.md", decoded.RunbookRef)
}

func TestTreasureChestItemsShow_NotFoundChecksBothKinds(t *testing.T) {
	dir := itemsTestRoot(t, oneProposedJewelYAML)
	writePotionsFile(t, dir, oneProposedPotionYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)

	err := treasureChestItemsShowCmd.RunE(treasureChestItemsShowCmd, []string{"no-such-item"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found (checked jewels and potions)")
}

func TestTreasureChestItemsAccept_PromotesPotion(t *testing.T) {
	dir := itemsTestRoot(t, oneProposedJewelYAML)
	writePotionsFile(t, dir, oneProposedPotionYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestItemsAcceptCmd.RunE(treasureChestItemsAcceptCmd, []string{"potion-1"}))
	})
	assert.Contains(t, out, "potion-1 -> status: accepted")
	assert.Contains(t, readPotionsFile(t, dir), "status: accepted")
}

func TestTreasureChestItemsDeprecate_PromotesPotion(t *testing.T) {
	dir := itemsTestRoot(t, oneProposedJewelYAML)
	writePotionsFile(t, dir, oneProposedPotionYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestItemsDeprecateCmd.RunE(treasureChestItemsDeprecateCmd, []string{"potion-1"}))
	})
	assert.Contains(t, out, "potion-1 -> status: deprecated")
	assert.Contains(t, readPotionsFile(t, dir), "status: deprecated")
}

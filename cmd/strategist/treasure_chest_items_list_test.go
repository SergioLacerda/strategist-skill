package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreasureChestItemsList_WithMissionRunDoesNotError(t *testing.T) {
	dir := itemsTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)
	attachMissionRun(t, treasureChestItemsListCmd)

	require.NoError(t, treasureChestItemsListCmd.RunE(treasureChestItemsListCmd, nil))
}

func TestTreasureChestItemsList_JewelLoadErrorPropagates(t *testing.T) {
	dir := itemsTestRoot(t, threeStatusJewelsYAML)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "jewels.yaml"), []byte("jewels: [unterminated\n"), 0o644))
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestItemsListCmd, "kind", "jewel")

	err := treasureChestItemsListCmd.RunE(treasureChestItemsListCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest items list")
}

func TestTreasureChestItemsList_PotionLoadErrorPropagates(t *testing.T) {
	dir := itemsTestRoot(t, threeStatusJewelsYAML)
	writePotionsFile(t, dir, "potions: [unterminated\n")
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestItemsListCmd, "kind", "potion")

	err := treasureChestItemsListCmd.RunE(treasureChestItemsListCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest items list")
}

func TestTreasureChestItemsList_DefaultExcludesDeprecated(t *testing.T) {
	dir := itemsTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestItemsListCmd.RunE(treasureChestItemsListCmd, nil))
	})
	assert.Contains(t, out, "jewel-proposed-1")
	assert.Contains(t, out, "jewel-accepted-1")
	assert.NotContains(t, out, "jewel-deprecated-1")
}

func TestTreasureChestItemsList_StatusAllIncludesDeprecated(t *testing.T) {
	dir := itemsTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestItemsListCmd, "status", "all")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestItemsListCmd.RunE(treasureChestItemsListCmd, nil))
	})
	assert.Contains(t, out, "jewel-proposed-1")
	assert.Contains(t, out, "jewel-accepted-1")
	assert.Contains(t, out, "jewel-deprecated-1")
}

func TestTreasureChestItemsList_ExplicitStatusFilter(t *testing.T) {
	dir := itemsTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestItemsListCmd, "status", "accepted")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestItemsListCmd.RunE(treasureChestItemsListCmd, nil))
	})
	assert.NotContains(t, out, "jewel-proposed-1")
	assert.Contains(t, out, "jewel-accepted-1")
	assert.NotContains(t, out, "jewel-deprecated-1")
}

func TestTreasureChestItemsList_ChestFilter(t *testing.T) {
	dir := itemsTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestItemsListCmd, "status", "all")
	setCmdFlag(t, treasureChestItemsListCmd, "chest", "nonexistent-chest")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestItemsListCmd.RunE(treasureChestItemsListCmd, nil))
	})
	assert.Contains(t, out, "no items match")
}

func TestTreasureChestItemsList_InvalidStatusRejected(t *testing.T) {
	dir := itemsTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestItemsListCmd, "status", "bogus")

	err := treasureChestItemsListCmd.RunE(treasureChestItemsListCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown --status "bogus"`)
}

func TestTreasureChestItemsList_InvalidKindRejected(t *testing.T) {
	dir := itemsTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestItemsListCmd, "kind", "bogus")

	err := treasureChestItemsListCmd.RunE(treasureChestItemsListCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown --kind "bogus"`)
}

func TestTreasureChestItemsList_KindJewelExcludesPotions(t *testing.T) {
	dir := itemsTestRoot(t, threeStatusJewelsYAML)
	writePotionsFile(t, dir, oneProposedPotionYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestItemsListCmd, "kind", "jewel")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestItemsListCmd.RunE(treasureChestItemsListCmd, nil))
	})
	assert.Contains(t, out, "jewel-proposed-1")
	assert.NotContains(t, out, "potion-1")
}

func TestTreasureChestItemsList_NoKindIncludesBothTypes(t *testing.T) {
	dir := itemsTestRoot(t, threeStatusJewelsYAML)
	writePotionsFile(t, dir, oneProposedPotionYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestItemsListCmd.RunE(treasureChestItemsListCmd, nil))
	})
	assert.Contains(t, out, "jewel-proposed-1")
	assert.Contains(t, out, "potion-1")
}

func TestTreasureChestItemsList_EmptyResultIsNotAnError(t *testing.T) {
	dir := itemsTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestItemsListCmd, "chest", "no-such-chest")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestItemsListCmd.RunE(treasureChestItemsListCmd, nil))
	})
	assert.Contains(t, out, "no items match the given filters")
}

func TestTreasureChestItemsList_JSONFormat(t *testing.T) {
	dir := itemsTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestItemsListCmd, "status", "all")
	setCmdFlag(t, treasureChestItemsListCmd, "format", "json")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestItemsListCmd.RunE(treasureChestItemsListCmd, nil))
	})
	var decoded []itemJSONEntry
	require.NoError(t, json.Unmarshal([]byte(out), &decoded))
	require.Len(t, decoded, 3)
	assert.Equal(t, "jewel-accepted-1", decoded[0].ID)
	assert.Equal(t, "accepted", decoded[0].Status)
	assert.Equal(t, "jewel", decoded[0].Kind)
}

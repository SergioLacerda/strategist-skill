package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreasureChestJewelList_DefaultExcludesDeprecated(t *testing.T) {
	dir := mineTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestJewelListCmd.RunE(treasureChestJewelListCmd, nil))
	})
	assert.Contains(t, out, "jewel-proposed-1")
	assert.Contains(t, out, "jewel-accepted-1")
	assert.NotContains(t, out, "jewel-deprecated-1")
}

func TestTreasureChestJewelList_StatusAllIncludesDeprecated(t *testing.T) {
	dir := mineTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestJewelListCmd, "status", "all")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestJewelListCmd.RunE(treasureChestJewelListCmd, nil))
	})
	assert.Contains(t, out, "jewel-proposed-1")
	assert.Contains(t, out, "jewel-accepted-1")
	assert.Contains(t, out, "jewel-deprecated-1")
}

func TestTreasureChestJewelList_ExplicitStatusFilter(t *testing.T) {
	dir := mineTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestJewelListCmd, "status", "accepted")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestJewelListCmd.RunE(treasureChestJewelListCmd, nil))
	})
	assert.NotContains(t, out, "jewel-proposed-1")
	assert.Contains(t, out, "jewel-accepted-1")
	assert.NotContains(t, out, "jewel-deprecated-1")
}

func TestTreasureChestJewelList_ChestFilter(t *testing.T) {
	dir := mineTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestJewelListCmd, "status", "all")
	setCmdFlag(t, treasureChestJewelListCmd, "chest", "nonexistent-chest")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestJewelListCmd.RunE(treasureChestJewelListCmd, nil))
	})
	assert.Contains(t, out, "no jewels match")
}

func TestTreasureChestJewelList_InvalidStatusRejected(t *testing.T) {
	dir := mineTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestJewelListCmd, "status", "bogus")

	err := treasureChestJewelListCmd.RunE(treasureChestJewelListCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown --status "bogus"`)
}

func TestTreasureChestJewelList_EmptyResultIsNotAnError(t *testing.T) {
	dir := mineTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestJewelListCmd, "chest", "no-such-chest")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestJewelListCmd.RunE(treasureChestJewelListCmd, nil))
	})
	assert.Contains(t, out, "no jewels match the given filters")
}

func TestTreasureChestJewelList_JSONFormat(t *testing.T) {
	dir := mineTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestJewelListCmd, "status", "all")
	setCmdFlag(t, treasureChestJewelListCmd, "format", "json")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestJewelListCmd.RunE(treasureChestJewelListCmd, nil))
	})
	var decoded []jewelJSONEntry
	require.NoError(t, json.Unmarshal([]byte(out), &decoded))
	require.Len(t, decoded, 3)
	assert.Equal(t, "jewel-accepted-1", decoded[0].ID)
	assert.Equal(t, "accepted", decoded[0].Status)
}

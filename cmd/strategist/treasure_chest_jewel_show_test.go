package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreasureChestJewelShow_Found(t *testing.T) {
	dir := mineTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestJewelShowCmd.RunE(treasureChestJewelShowCmd, []string{"jewel-accepted-1"}))
	})
	assert.Contains(t, out, "jewel-accepted-1")
	assert.Contains(t, out, "Accepted jewel statement.")
	assert.Contains(t, out, "source#b")
	assert.Contains(t, out, "accepted")
}

func TestTreasureChestJewelShow_NotFound(t *testing.T) {
	dir := mineTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	setTreasureChestRoot(t, dir)

	err := treasureChestJewelShowCmd.RunE(treasureChestJewelShowCmd, []string{"no-such-jewel"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `jewel "no-such-jewel" not found`)
}

func TestTreasureChestJewelShow_JSONFormat(t *testing.T) {
	dir := mineTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestJewelShowCmd, "format", "json")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestJewelShowCmd.RunE(treasureChestJewelShowCmd, []string{"jewel-accepted-1"}))
	})
	var decoded jsonJewelShowEntry
	require.NoError(t, json.Unmarshal([]byte(out), &decoded))
	assert.Equal(t, "jewel-accepted-1", decoded.ID)
	assert.Equal(t, []string{"source#b"}, decoded.SourceRefs)
}

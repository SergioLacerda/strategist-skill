package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreasureChestMine_ListShowsOnlyProposed(t *testing.T) {
	dir := mineTestRoot(t, `
schema_version: "1"
jewels:
  - id: jewel-1
    chest_id: source
    kind: pattern
    statement: "Widgets require explicit teardown."
    source_refs: ["source#widgets"]
    trust: T1
    status: proposed
    reviewed_by: agent
  - id: jewel-2
    chest_id: source
    kind: decision
    statement: "Already accepted fact."
    source_refs: ["source#x"]
    trust: T1
    status: accepted
    reviewed_by: human
`)
	resetTreasureChestFlags(t)
	resetTreasureChestMineFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestMineCmd, "list", "true")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestMineCmd.RunE(treasureChestMineCmd, nil))
	})
	assert.Contains(t, out, "jewel-1")
	assert.NotContains(t, out, "jewel-2")
}

func TestTreasureChestMine_ListJSONFormat(t *testing.T) {
	dir := mineTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestMineFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestMineCmd, "list", "true")
	setCmdFlag(t, treasureChestMineCmd, "format", "json")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestMineCmd.RunE(treasureChestMineCmd, nil))
	})
	assert.Contains(t, out, `"id": "jewel-1"`)
	assert.Contains(t, out, `"status": "proposed"`)
}

func TestTreasureChestMine_ListEmptyIsNotError(t *testing.T) {
	dir := mineTestRoot(t, "schema_version: \"1\"\njewels: []\n")
	resetTreasureChestFlags(t)
	resetTreasureChestMineFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestMineCmd, "list", "true")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestMineCmd.RunE(treasureChestMineCmd, nil))
	})
	assert.Contains(t, out, "no proposed jewels")
}

package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const threeStatusJewelsYAML = `
schema_version: "1"
jewels:
  - id: jewel-proposed-1
    chest_id: source
    kind: pattern
    statement: "Proposed jewel statement."
    source_refs: ["source#a"]
    trust: T1
    status: proposed
    reviewed_by: agent
    score: { value: 40, reasons: ["seen once"] }
  - id: jewel-accepted-1
    chest_id: source
    kind: gap
    statement: "Accepted jewel statement."
    source_refs: ["source#b"]
    trust: T1
    status: accepted
    reviewed_by: human
    last_reviewed: "2026-07-15"
    score: { value: 60, reasons: ["seen twice"] }
  - id: jewel-deprecated-1
    chest_id: source
    kind: gap
    statement: "Deprecated jewel statement."
    source_refs: ["source#c"]
    trust: T1
    status: deprecated
    reviewed_by: human
    last_reviewed: "2026-07-10"
    score: { value: 10, reasons: ["stale"] }
`

func resetTreasureChestJewelFlags(t *testing.T) {
	t.Helper()
	origStatus := treasureChestJewelStatus
	origChest := treasureChestJewelChest
	origFormat := treasureChestJewelFormat
	t.Cleanup(func() {
		treasureChestJewelStatus = origStatus
		treasureChestJewelChest = origChest
		treasureChestJewelFormat = origFormat
	})
	treasureChestJewelStatus = ""
	treasureChestJewelChest = ""
	treasureChestJewelFormat = "table"
}

func TestTreasureChestJewelList_DefaultExcludesDeprecated(t *testing.T) {
	dir := mineTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	treasureChestRoot = dir

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
	treasureChestRoot = dir
	treasureChestJewelStatus = "all"

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
	treasureChestRoot = dir
	treasureChestJewelStatus = "accepted"

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
	treasureChestRoot = dir
	treasureChestJewelStatus = "all"
	treasureChestJewelChest = "nonexistent-chest"

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestJewelListCmd.RunE(treasureChestJewelListCmd, nil))
	})
	assert.Contains(t, out, "no jewels match")
}

func TestTreasureChestJewelList_InvalidStatusRejected(t *testing.T) {
	dir := mineTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	treasureChestRoot = dir
	treasureChestJewelStatus = "bogus"

	err := treasureChestJewelListCmd.RunE(treasureChestJewelListCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown --status "bogus"`)
}

func TestTreasureChestJewelList_EmptyResultIsNotAnError(t *testing.T) {
	dir := mineTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	treasureChestRoot = dir
	treasureChestJewelChest = "no-such-chest"

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestJewelListCmd.RunE(treasureChestJewelListCmd, nil))
	})
	assert.Contains(t, out, "no jewels match the given filters")
}

func TestTreasureChestJewelList_JSONFormat(t *testing.T) {
	dir := mineTestRoot(t, threeStatusJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	treasureChestRoot = dir
	treasureChestJewelStatus = "all"
	treasureChestJewelFormat = "json"

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestJewelListCmd.RunE(treasureChestJewelListCmd, nil))
	})
	var decoded []jsonJewelListEntry
	require.NoError(t, json.Unmarshal([]byte(out), &decoded))
	require.Len(t, decoded, 3)
	assert.Equal(t, "jewel-accepted-1", decoded[0].ID)
	assert.Equal(t, "accepted", decoded[0].Status)
}

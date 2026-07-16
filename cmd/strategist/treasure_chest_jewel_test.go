package main

import (
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

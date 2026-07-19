package main

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	origStatus := cmdFlagString(t, treasureChestJewelListCmd, "status")
	origChest := cmdFlagString(t, treasureChestJewelListCmd, "chest")
	origListFormat := cmdFlagString(t, treasureChestJewelListCmd, "format")
	origShowFormat := cmdFlagString(t, treasureChestJewelShowCmd, "format")
	t.Cleanup(func() {
		setCmdFlag(t, treasureChestJewelListCmd, "status", origStatus)
		setCmdFlag(t, treasureChestJewelListCmd, "chest", origChest)
		setCmdFlag(t, treasureChestJewelListCmd, "format", origListFormat)
		setCmdFlag(t, treasureChestJewelShowCmd, "format", origShowFormat)
	})
	setCmdFlag(t, treasureChestJewelListCmd, "status", "")
	setCmdFlag(t, treasureChestJewelListCmd, "chest", "")
	setCmdFlag(t, treasureChestJewelListCmd, "format", "table")
	setCmdFlag(t, treasureChestJewelShowCmd, "format", "table")
}

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
	var decoded []jsonJewelListEntry
	require.NoError(t, json.Unmarshal([]byte(out), &decoded))
	require.Len(t, decoded, 3)
	assert.Equal(t, "jewel-accepted-1", decoded[0].ID)
	assert.Equal(t, "accepted", decoded[0].Status)
}

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

// --- curation subcommands (Track: treasure-chest-jewel-mine-consolidation, GAP-TC-01) ---

func resetTreasureChestJewelEvidence(t *testing.T) {
	t.Helper()
	orig := cmdFlagString(t, treasureChestJewelVerifyCmd, "evidence")
	t.Cleanup(func() {
		setCmdFlag(t, treasureChestJewelVerifyCmd, "evidence", orig)
	})
	setCmdFlag(t, treasureChestJewelVerifyCmd, "evidence", "")
}

func readJewelsFile(t *testing.T, dir string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "jewels.yaml"))
	require.NoError(t, err)
	return string(raw)
}

func TestTreasureChestJewelAccept_PromotesProposedJewel(t *testing.T) {
	dir := mineTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestJewelAcceptCmd.RunE(treasureChestJewelAcceptCmd, []string{"jewel-1"}))
	})
	assert.Contains(t, out, "jewel-1 -> status: accepted")
	assert.Contains(t, readJewelsFile(t, dir), "status: accepted")
}

func TestTreasureChestJewelAccept_MultipleIDs(t *testing.T) {
	dir := mineTestRoot(t, twoProposedJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestJewelAcceptCmd.RunE(treasureChestJewelAcceptCmd, []string{"jewel-1", "jewel-2"}))
	})
	assert.Contains(t, out, "jewel-1 -> status: accepted")
	assert.Contains(t, out, "jewel-2 -> status: accepted")
	assert.Equal(t, 2, countOccurrences(readJewelsFile(t, dir), "reviewed_by: human"))
}

func TestTreasureChestJewelVerify_RequiresEvidence(t *testing.T) {
	dir := mineTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	resetTreasureChestJewelEvidence(t)
	setTreasureChestRoot(t, dir)

	err := treasureChestJewelVerifyCmd.RunE(treasureChestJewelVerifyCmd, []string{"jewel-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--evidence is required")
}

func TestTreasureChestJewelVerify_RecordsEvidence(t *testing.T) {
	dir := mineTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	resetTreasureChestJewelEvidence(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestJewelVerifyCmd, "evidence", "missions/m-42/outcome.md")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestJewelVerifyCmd.RunE(treasureChestJewelVerifyCmd, []string{"jewel-1"}))
	})
	assert.Contains(t, out, "jewel-1 -> status: verified")
	content := readJewelsFile(t, dir)
	assert.Contains(t, content, "status: verified")
	assert.Contains(t, content, "missions/m-42/outcome.md")
}

func TestTreasureChestJewelDeprecate_MarksJewelDeprecated(t *testing.T) {
	dir := mineTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestJewelDeprecateCmd.RunE(treasureChestJewelDeprecateCmd, []string{"jewel-1"}))
	})
	assert.Contains(t, out, "jewel-1 -> status: deprecated")
	assert.Contains(t, readJewelsFile(t, dir), "status: deprecated")
}

func TestTreasureChestJewelMigrateStatus_RewritesLegacyActive(t *testing.T) {
	dir := mineTestRoot(t, `
schema_version: "1"
jewels:
  - id: jewel-legacy
    chest_id: source
    kind: pattern
    statement: "Legacy jewel."
    source_refs: ["source#x"]
    trust: T1
    status: active
    reviewed_by: agent
`)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestJewelMigrateStatusCmd.RunE(treasureChestJewelMigrateStatusCmd, nil))
	})
	assert.Contains(t, out, "1 jewel(s) migrated")
	content := readJewelsFile(t, dir)
	assert.NotContains(t, content, "status: active")
	assert.Contains(t, content, "status: accepted")
}

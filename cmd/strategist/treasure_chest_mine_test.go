package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetTreasureChestMineFlags(t *testing.T) {
	t.Helper()
	origList := treasureChestMineList
	origFormat := treasureChestMineFormat
	origAccept := treasureChestMineAccept
	origVerify := treasureChestMineVerify
	origEvidence := treasureChestMineEvidence
	origDeprecate := treasureChestMineDeprecate
	origMigrate := treasureChestMineMigrateStatus
	t.Cleanup(func() {
		treasureChestMineList = origList
		treasureChestMineFormat = origFormat
		treasureChestMineAccept = origAccept
		treasureChestMineVerify = origVerify
		treasureChestMineEvidence = origEvidence
		treasureChestMineDeprecate = origDeprecate
		treasureChestMineMigrateStatus = origMigrate
	})
	treasureChestMineList = false
	treasureChestMineFormat = "table"
	treasureChestMineAccept = ""
	treasureChestMineVerify = ""
	treasureChestMineEvidence = ""
	treasureChestMineDeprecate = ""
	treasureChestMineMigrateStatus = false
}

func mineTestRoot(t *testing.T, jewelsYAML string) string {
	t.Helper()
	dir := minimalTreasureChestRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "jewels.yaml"), []byte(jewelsYAML), 0o644))
	return dir
}

const oneProposedJewelYAML = `
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
    score:
      value: 55
      reasons: ["recurring across 2 missions"]
`

func TestTreasureChestMine_RequiresExactlyOneAction(t *testing.T) {
	dir := mineTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestMineFlags(t)
	treasureChestRoot = dir

	err := treasureChestMineCmd.RunE(treasureChestMineCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "specify exactly one of")
}

func TestTreasureChestMine_VerifyRequiresEvidence(t *testing.T) {
	dir := mineTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestMineFlags(t)
	treasureChestRoot = dir
	treasureChestMineVerify = "jewel-1"

	err := treasureChestMineCmd.RunE(treasureChestMineCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--verify requires --evidence")
}

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
	treasureChestRoot = dir
	treasureChestMineList = true

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
	treasureChestRoot = dir
	treasureChestMineList = true
	treasureChestMineFormat = "json"

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
	treasureChestRoot = dir
	treasureChestMineList = true

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestMineCmd.RunE(treasureChestMineCmd, nil))
	})
	assert.Contains(t, out, "no proposed jewels")
}

func TestTreasureChestMine_Accept(t *testing.T) {
	dir := mineTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestMineFlags(t)
	treasureChestRoot = dir
	treasureChestMineAccept = "jewel-1"

	require.NoError(t, treasureChestMineCmd.RunE(treasureChestMineCmd, nil))

	raw, err := os.ReadFile(filepath.Join(dir, "jewels.yaml"))
	require.NoError(t, err)
	content := string(raw)
	assert.Contains(t, content, "status: accepted")
	assert.Contains(t, content, "reviewed_by: human")
}

func TestTreasureChestMine_VerifyRecordsEvidence(t *testing.T) {
	dir := mineTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestMineFlags(t)
	treasureChestRoot = dir
	treasureChestMineVerify = "jewel-1"
	treasureChestMineEvidence = "dojo-run-42"

	require.NoError(t, treasureChestMineCmd.RunE(treasureChestMineCmd, nil))

	raw, err := os.ReadFile(filepath.Join(dir, "jewels.yaml"))
	require.NoError(t, err)
	content := string(raw)
	assert.Contains(t, content, "status: verified")
	assert.Contains(t, content, "dojo-run-42")
}

func TestTreasureChestMine_Deprecate(t *testing.T) {
	dir := mineTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestMineFlags(t)
	treasureChestRoot = dir
	treasureChestMineDeprecate = "jewel-1"

	require.NoError(t, treasureChestMineCmd.RunE(treasureChestMineCmd, nil))

	raw, err := os.ReadFile(filepath.Join(dir, "jewels.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "status: deprecated")
}

func TestTreasureChestMine_CannotPromoteDeprecated(t *testing.T) {
	dir := mineTestRoot(t, `
schema_version: "1"
jewels:
  - id: jewel-1
    chest_id: source
    kind: pattern
    statement: "Stale fact."
    source_refs: ["source#x"]
    trust: T1
    status: deprecated
    reviewed_by: human
`)
	resetTreasureChestFlags(t)
	resetTreasureChestMineFlags(t)
	treasureChestRoot = dir
	treasureChestMineAccept = "jewel-1"

	err := treasureChestMineCmd.RunE(treasureChestMineCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deprecated")
}

func TestTreasureChestMine_UnknownIDErrors(t *testing.T) {
	dir := mineTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestMineFlags(t)
	treasureChestRoot = dir
	treasureChestMineAccept = "does-not-exist"

	err := treasureChestMineCmd.RunE(treasureChestMineCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestTreasureChestMine_MigrateStatus(t *testing.T) {
	dir := mineTestRoot(t, `
schema_version: "1"
jewels:
  - id: jewel-1
    chest_id: source
    statement: "Pre-migration jewel."
    source_refs: ["source#x"]
    trust: T1
    status: active
    reviewed_by: agent
  - id: jewel-2
    chest_id: source
    statement: "Already migrated."
    source_refs: ["source#y"]
    trust: T1
    status: accepted
    reviewed_by: human
`)
	resetTreasureChestFlags(t)
	resetTreasureChestMineFlags(t)
	treasureChestRoot = dir
	treasureChestMineMigrateStatus = true

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestMineCmd.RunE(treasureChestMineCmd, nil))
	})
	assert.Contains(t, out, "1 jewel(s) migrated")

	raw, err := os.ReadFile(filepath.Join(dir, "jewels.yaml"))
	require.NoError(t, err)
	content := string(raw)
	assert.NotContains(t, content, "status: active")

	// jewels.yaml must now load cleanly (loadJewels rejects legacy "active").
	_, err = loadJewels(dir, nil)
	require.NoError(t, err)
}

func TestTreasureChestMine_MigrateStatusNoLegacyEntries(t *testing.T) {
	dir := mineTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestMineFlags(t)
	treasureChestRoot = dir
	treasureChestMineMigrateStatus = true

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestMineCmd.RunE(treasureChestMineCmd, nil))
	})
	assert.Contains(t, out, "nothing to migrate")
}

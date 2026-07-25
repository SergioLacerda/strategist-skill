package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetTreasureChestMineFlags(t *testing.T) {
	t.Helper()
	origList := cmdFlagBool(t, treasureChestMineCmd, "list")
	origFormat := cmdFlagString(t, treasureChestMineCmd, "format")
	origAccept := cmdFlagString(t, treasureChestMineCmd, "accept")
	origVerify := cmdFlagString(t, treasureChestMineCmd, "verify")
	origEvidence := cmdFlagString(t, treasureChestMineCmd, "evidence")
	origDeprecate := cmdFlagString(t, treasureChestMineCmd, "deprecate")
	origMigrate := cmdFlagBool(t, treasureChestMineCmd, "migrate-status")
	t.Cleanup(func() {
		setCmdFlag(t, treasureChestMineCmd, "list", fmt.Sprint(origList))
		setCmdFlag(t, treasureChestMineCmd, "format", origFormat)
		setCmdFlag(t, treasureChestMineCmd, "accept", origAccept)
		setCmdFlag(t, treasureChestMineCmd, "verify", origVerify)
		setCmdFlag(t, treasureChestMineCmd, "evidence", origEvidence)
		setCmdFlag(t, treasureChestMineCmd, "deprecate", origDeprecate)
		setCmdFlag(t, treasureChestMineCmd, "migrate-status", fmt.Sprint(origMigrate))
	})
	setCmdFlag(t, treasureChestMineCmd, "list", "false")
	setCmdFlag(t, treasureChestMineCmd, "format", "table")
	setCmdFlag(t, treasureChestMineCmd, "accept", "")
	setCmdFlag(t, treasureChestMineCmd, "verify", "")
	setCmdFlag(t, treasureChestMineCmd, "evidence", "")
	setCmdFlag(t, treasureChestMineCmd, "deprecate", "")
	setCmdFlag(t, treasureChestMineCmd, "migrate-status", "false")
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

const twoProposedJewelsYAML = `
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
  - id: jewel-2
    chest_id: source
    kind: pattern
    statement: "Widgets need stable cache keys."
    source_refs: ["source#cache"]
    trust: T1
    status: proposed
    reviewed_by: agent
    score:
      value: 45
      reasons: ["recurring across 2 missions"]
`

func TestTreasureChestMine_RequiresExactlyOneAction(t *testing.T) {
	dir := mineTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestMineFlags(t)
	setTreasureChestRoot(t, dir)

	err := treasureChestMineCmd.RunE(treasureChestMineCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "specify exactly one of")
}

func TestTreasureChestMine_VerifyRequiresEvidence(t *testing.T) {
	dir := mineTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestMineFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestMineCmd, "verify", "jewel-1")

	err := treasureChestMineCmd.RunE(treasureChestMineCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--verify requires --evidence")
}

func TestTreasureChestMine_Accept(t *testing.T) {
	dir := mineTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestMineFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestMineCmd, "accept", "jewel-1")

	require.NoError(t, treasureChestMineCmd.RunE(treasureChestMineCmd, nil))

	raw, err := os.ReadFile(filepath.Join(dir, "jewels.yaml"))
	require.NoError(t, err)
	content := string(raw)
	assert.Contains(t, content, "status: accepted")
	assert.Contains(t, content, "reviewed_by: human")
	assert.Contains(t, content, "history:")
	assert.Contains(t, content, "by: human")
}

func TestTreasureChestMine_AcceptCommaSeparatedIDs(t *testing.T) {
	dir := mineTestRoot(t, twoProposedJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestMineFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestMineCmd, "accept", "jewel-1, jewel-2")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestMineCmd.RunE(treasureChestMineCmd, nil))
	})
	assert.Contains(t, out, "jewel-1 -> status: accepted")
	assert.Contains(t, out, "jewel-2 -> status: accepted")
	assert.Equal(t, 2, countOccurrences(readJewelsFile(t, dir), "reviewed_by: human"))
}

func TestTreasureChestMine_VerifyRecordsEvidence(t *testing.T) {
	dir := mineTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestMineFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestMineCmd, "verify", "jewel-1")
	setCmdFlag(t, treasureChestMineCmd, "evidence", "dojo-run-42")

	require.NoError(t, treasureChestMineCmd.RunE(treasureChestMineCmd, nil))

	raw, err := os.ReadFile(filepath.Join(dir, "jewels.yaml"))
	require.NoError(t, err)
	content := string(raw)
	assert.Contains(t, content, "status: verified")
	assert.Contains(t, content, "dojo-run-42")
	assert.Contains(t, content, "history:")
	assert.Contains(t, content, "evidence_ref: dojo-run-42")
}

func TestTreasureChestMine_Deprecate(t *testing.T) {
	dir := mineTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestMineFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestMineCmd, "deprecate", "jewel-1")

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
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestMineCmd, "accept", "jewel-1")

	err := treasureChestMineCmd.RunE(treasureChestMineCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deprecated")
}

func TestTreasureChestMine_UnknownIDErrors(t *testing.T) {
	dir := mineTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestMineFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestMineCmd, "accept", "does-not-exist")

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
    kind: pattern
    statement: "Pre-migration jewel."
    source_refs: ["source#x"]
    trust: T1
    status: active
    reviewed_by: agent
  - id: jewel-2
    chest_id: source
    kind: pattern
    statement: "Already migrated."
    source_refs: ["source#y"]
    trust: T1
    status: accepted
    reviewed_by: human
`)
	resetTreasureChestFlags(t)
	resetTreasureChestMineFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestMineCmd, "migrate-status", "true")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestMineCmd.RunE(treasureChestMineCmd, nil))
	})
	assert.Contains(t, out, "1 jewel(s) migrated")

	raw, err := os.ReadFile(filepath.Join(dir, "jewels.yaml"))
	require.NoError(t, err)
	content := string(raw)
	assert.NotContains(t, content, "status: active")

	// jewels.yaml must now load cleanly (treasure.LoadJewels rejects legacy "active").
	_, err = treasure.LoadJewels(dir, nil)
	require.NoError(t, err)
}

func TestTreasureChestMine_MigrateStatusNoLegacyEntries(t *testing.T) {
	dir := mineTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestMineFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestMineCmd, "migrate-status", "true")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestMineCmd.RunE(treasureChestMineCmd, nil))
	})
	assert.Contains(t, out, "nothing to migrate")
}

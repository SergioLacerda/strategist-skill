package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const evidenceQualityJewelsYAML = `
schema_version: "1"
jewels:
  - id: jewel-expired
    chest_id: source
    kind: gap
    statement: "This jewel expired."
    source_refs: ["source#a"]
    trust: T1
    status: accepted
    reviewed_by: human
    valid_until: "2020-01-01T00:00:00Z"
  - id: jewel-dup-1
    chest_id: source
    kind: pattern
    statement: "Widgets require explicit teardown."
    source_refs: ["source#b"]
    trust: T1
    status: accepted
    reviewed_by: human
  - id: jewel-dup-2
    chest_id: source
    kind: pattern
    statement: "widgets require explicit teardown."
    source_refs: ["source#c"]
    trust: T1
    status: accepted
    reviewed_by: human
  - id: jewel-conflict-a
    chest_id: source
    kind: decision
    statement: "The default is enabled."
    source_refs: ["source#shared"]
    trust: T1
    status: accepted
    reviewed_by: human
  - id: jewel-conflict-b
    chest_id: source
    kind: decision
    statement: "The default is disabled."
    source_refs: ["source#shared"]
    trust: T1
    status: accepted
    reviewed_by: human
`

func TestTreasureChestItemsCheckEvidence_ReportsAllThreeFindings(t *testing.T) {
	dir := itemsTestRoot(t, evidenceQualityJewelsYAML)
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestItemsCheckEvidenceCmd.RunE(treasureChestItemsCheckEvidenceCmd, []string{"source"}))
	})

	assert.Contains(t, out, "expired: jewel-expired")
	assert.Contains(t, out, "duplicate: jewel-dup-1 ~ jewel-dup-2")
	assert.Contains(t, out, "conflict: jewel-conflict-a <> jewel-conflict-b")
}

func TestTreasureChestItemsCheckEvidence_NoFindingsExitsCleanly(t *testing.T) {
	dir := itemsTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestItemsCheckEvidenceCmd.RunE(treasureChestItemsCheckEvidenceCmd, []string{"source"}))
	})

	assert.Contains(t, out, "no findings")
}

func TestTreasureChestItemsCheckEvidence_UnknownChestExitsCleanly(t *testing.T) {
	dir := itemsTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestItemsCheckEvidenceCmd.RunE(treasureChestItemsCheckEvidenceCmd, []string{"no-such-chest"}))
	})

	assert.Contains(t, out, "no findings")
}

func TestTreasureChestItemsCheckEvidence_WithMissionRunDoesNotError(t *testing.T) {
	dir := itemsTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)
	attachMissionRun(t, treasureChestItemsCheckEvidenceCmd)

	require.NoError(t, treasureChestItemsCheckEvidenceCmd.RunE(treasureChestItemsCheckEvidenceCmd, []string{"source"}))
}

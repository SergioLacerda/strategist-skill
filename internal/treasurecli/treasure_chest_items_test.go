package treasurecli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
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

// itemsTestRoot returns a minimal .strategist/ root with jewelsYAML written to jewels.yaml.
func itemsTestRoot(t *testing.T, jewelsYAML string) string {
	t.Helper()
	dir := minimalTreasureChestRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "jewels.yaml"), []byte(jewelsYAML), 0o644))
	return dir
}

func resetTreasureChestItemsFlags(t *testing.T) {
	t.Helper()
	origStatus := cmdFlagString(t, treasureChestItemsListCmd, "status")
	origChest := cmdFlagString(t, treasureChestItemsListCmd, "chest")
	origKind := cmdFlagString(t, treasureChestItemsListCmd, "kind")
	origListFormat := cmdFlagString(t, treasureChestItemsListCmd, "format")
	origShowFormat := cmdFlagString(t, treasureChestItemsShowCmd, "format")
	t.Cleanup(func() {
		setCmdFlag(t, treasureChestItemsListCmd, "status", origStatus)
		setCmdFlag(t, treasureChestItemsListCmd, "chest", origChest)
		setCmdFlag(t, treasureChestItemsListCmd, "kind", origKind)
		setCmdFlag(t, treasureChestItemsListCmd, "format", origListFormat)
		setCmdFlag(t, treasureChestItemsShowCmd, "format", origShowFormat)
	})
	setCmdFlag(t, treasureChestItemsListCmd, "status", "")
	setCmdFlag(t, treasureChestItemsListCmd, "chest", "")
	setCmdFlag(t, treasureChestItemsListCmd, "kind", "")
	setCmdFlag(t, treasureChestItemsListCmd, "format", "table")
	setCmdFlag(t, treasureChestItemsShowCmd, "format", "table")
}

func resetTreasureChestItemsEvidence(t *testing.T) {
	t.Helper()
	orig := cmdFlagString(t, treasureChestItemsVerifyCmd, "evidence")
	t.Cleanup(func() {
		setCmdFlag(t, treasureChestItemsVerifyCmd, "evidence", orig)
	})
	setCmdFlag(t, treasureChestItemsVerifyCmd, "evidence", "")
}

func readJewelsFile(t *testing.T, dir string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "jewels.yaml"))
	require.NoError(t, err)
	return string(raw)
}

func TestTreasureChestItemsAccept_WithMissionRunDoesNotError(t *testing.T) {
	dir := itemsTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)
	attachMissionRun(t, treasureChestItemsAcceptCmd)

	require.NoError(t, treasureChestItemsAcceptCmd.RunE(treasureChestItemsAcceptCmd, []string{"jewel-1"}))
}

func TestTreasureChestItemsAccept_ResolveRootError(t *testing.T) {
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, "")
	chdirForTest(t, t.TempDir())

	err := treasureChestItemsAcceptCmd.RunE(treasureChestItemsAcceptCmd, []string{"jewel-1"})
	require.Error(t, err)
}

func TestLoadJewelsForCmd_ResolveRootError(t *testing.T) {
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, "") // force cwd-based discovery
	chdirForTest(t, t.TempDir())

	_, err := loadJewelsForCmd(treasureChestItemsListCmd, "treasure-chest items list")
	require.Error(t, err)
}

func TestLoadPotionsForCmd_ResolveRootError(t *testing.T) {
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, "") // force cwd-based discovery
	chdirForTest(t, t.TempDir())

	_, err := loadPotionsForCmd(treasureChestItemsListCmd, "treasure-chest items list")
	require.Error(t, err)
}

func TestLoadPotionsForCmd_InvalidPotionsYAML(t *testing.T) {
	dir := itemsTestRoot(t, oneProposedJewelYAML)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "potions.yaml"), []byte("potions: [unterminated\n"), 0o644))
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	_, err := loadPotionsForCmd(treasureChestItemsListCmd, "treasure-chest items list")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest items list")
}

func TestBestEffortGoverned_InvalidYAMLReturnsNil(t *testing.T) {
	dir := itemsTestRoot(t, oneProposedJewelYAML)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte("chests: [unterminated\n"), 0o644))

	assert.Nil(t, bestEffortGoverned(dir))
}

func TestPromoteItem_PotionDeprecatedCannotPromote(t *testing.T) {
	dir := itemsTestRoot(t, oneProposedJewelYAML)
	writePotionsFile(t, dir, `
schema_version: "1"
potions:
  - id: potion-deprecated-1
    chest_id: runbooks
    runbook_ref: docs/runbooks/sample.md
    when_to_use: "When sample breaks."
    trust: T2
    status: deprecated
    source_refs: ["docs/runbooks/sample.md"]
    reviewed_by: human
`)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)

	err := treasureChestItemsAcceptCmd.RunE(treasureChestItemsAcceptCmd, []string{"potion-deprecated-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "promote potion")
	assert.Contains(t, err.Error(), "deprecated")
}

func TestTreasureChestItemsMigrateStatus_Error(t *testing.T) {
	dir := itemsTestRoot(t, oneProposedJewelYAML)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "jewels.yaml"), []byte("jewels: [unterminated\n"), 0o644))
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	err := runTreasureChestItemsMigrateStatus(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest items")
}

func TestTreasureChestItemsAccept_PromotesProposedJewel(t *testing.T) {
	dir := itemsTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestItemsAcceptCmd.RunE(treasureChestItemsAcceptCmd, []string{"jewel-1"}))
	})
	assert.Contains(t, out, "jewel-1 -> status: accepted")
	assert.Contains(t, readJewelsFile(t, dir), "status: accepted")
}

func TestTreasureChestItemsAccept_MultipleIDs(t *testing.T) {
	dir := itemsTestRoot(t, twoProposedJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestItemsAcceptCmd.RunE(treasureChestItemsAcceptCmd, []string{"jewel-1", "jewel-2"}))
	})
	assert.Contains(t, out, "jewel-1 -> status: accepted")
	assert.Contains(t, out, "jewel-2 -> status: accepted")
	assert.Equal(t, 2, countOccurrences(readJewelsFile(t, dir), "reviewed_by: human"))
}

func TestTreasureChestItemsAccept_CommaSeparatedIDs(t *testing.T) {
	dir := itemsTestRoot(t, twoProposedJewelsYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestItemsAcceptCmd.RunE(treasureChestItemsAcceptCmd, []string{"jewel-1, jewel-2"}))
	})
	assert.Contains(t, out, "jewel-1 -> status: accepted")
	assert.Contains(t, out, "jewel-2 -> status: accepted")
	assert.Equal(t, 2, countOccurrences(readJewelsFile(t, dir), "reviewed_by: human"))
}

func TestTreasureChestItemsVerify_RequiresEvidence(t *testing.T) {
	dir := itemsTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	resetTreasureChestItemsEvidence(t)
	setTreasureChestRoot(t, dir)

	err := treasureChestItemsVerifyCmd.RunE(treasureChestItemsVerifyCmd, []string{"jewel-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--evidence is required")
}

func TestTreasureChestItemsVerify_RecordsEvidence(t *testing.T) {
	dir := itemsTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	resetTreasureChestItemsEvidence(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestItemsVerifyCmd, "evidence", "missions/m-42/outcome.md")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestItemsVerifyCmd.RunE(treasureChestItemsVerifyCmd, []string{"jewel-1"}))
	})
	assert.Contains(t, out, "jewel-1 -> status: verified")
	content := readJewelsFile(t, dir)
	assert.Contains(t, content, "status: verified")
	assert.Contains(t, content, "missions/m-42/outcome.md")
}

func TestTreasureChestItemsDeprecate_MarksJewelDeprecated(t *testing.T) {
	dir := itemsTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestItemsDeprecateCmd.RunE(treasureChestItemsDeprecateCmd, []string{"jewel-1"}))
	})
	assert.Contains(t, out, "jewel-1 -> status: deprecated")
	assert.Contains(t, readJewelsFile(t, dir), "status: deprecated")
}

func TestTreasureChestItemsDeprecate_CannotPromoteDeprecated(t *testing.T) {
	dir := itemsTestRoot(t, `
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
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)

	err := treasureChestItemsAcceptCmd.RunE(treasureChestItemsAcceptCmd, []string{"jewel-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deprecated")
}

func TestTreasureChestItemsPromote_UnknownIDErrors(t *testing.T) {
	dir := itemsTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)

	err := treasureChestItemsAcceptCmd.RunE(treasureChestItemsAcceptCmd, []string{"does-not-exist"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found (checked jewels and potions)")
}

func TestTreasureChestItemsMigrateStatus_RewritesLegacyActive(t *testing.T) {
	dir := itemsTestRoot(t, `
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
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestItemsMigrateStatusCmd.RunE(treasureChestItemsMigrateStatusCmd, nil))
	})
	assert.Contains(t, out, "1 jewel(s) migrated")
	content := readJewelsFile(t, dir)
	assert.NotContains(t, content, "status: active")
	assert.Contains(t, content, "status: accepted")

	// jewels.yaml must now load cleanly (treasure.LoadJewels rejects legacy "active").
	_, err := treasure.LoadJewels(dir, nil)
	require.NoError(t, err)
}

func TestTreasureChestItemsMigrateStatus_NoLegacyEntries(t *testing.T) {
	dir := itemsTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestItemsMigrateStatusCmd.RunE(treasureChestItemsMigrateStatusCmd, nil))
	})
	assert.Contains(t, out, "nothing to migrate")
}

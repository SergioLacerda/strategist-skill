package treasurecli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- status (read-only) tests ---

func TestTreasureChestCmd_ShowsChestsSection(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)
	setTreasureChestDoIndex(t, false)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	})
	assert.Contains(t, out, "CHESTS")
	assert.Contains(t, out, "source")
	assert.Contains(t, out, "INDEX")
}

func TestTreasureChestListCmd_ShowsStatus(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestListCmd.RunE(treasureChestListCmd, nil))
	})
	assert.Contains(t, out, "CHESTS")
}

func TestTreasureChestCmd_WithMissionRunDoesNotError(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)
	attachMissionRun(t, treasureChestCmd)

	require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
}

func TestTreasureChestCmd_ShowsTrustTierFromGoverned(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)
	setTreasureChestDoIndex(t, false)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	})
	assert.Contains(t, out, "T1")
	assert.Contains(t, out, "fresh") // last_reviewed is set
}

func TestTreasureChestCmd_DefaultIsReadOnly(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)
	setTreasureChestDoIndex(t, false)

	require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))

	// Default invocation must not produce a compiled artifact.
	assert.NoFileExists(t, filepath.Join(dir, ".compiled", ".index.gz"))
}

func TestTreasureChestCmd_SuggestsIndexWhenCompiledAbsent(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)
	setTreasureChestDoIndex(t, false)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	})
	assert.Contains(t, out, "absent")
	assert.Contains(t, out, "--index")
}

func TestTreasureChestCmd_ShowsDriftMissingGovernance(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: epic
base_path: .analysis
slots:
  discovery: brainstorming
  refinement: openspec-explore
  execution: sdd-ask
treasure_chests:
  - id: undeclared-chest
    path: .some/path
    scope: all
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte("schema_version: \"1\"\nchests: []\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "knowledge.index.yaml"), []byte("sources: []\n"), 0o644))

	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)
	setTreasureChestDoIndex(t, false)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	})
	assert.Contains(t, out, "missing_governance")
	assert.Contains(t, out, "drift")
}

func TestTreasureChestCmd_ShowsDriftMissingIndex(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: epic
base_path: .analysis
slots:
  discovery: brainstorming
  refinement: openspec-explore
  execution: sdd-ask
treasure_chests:
  - id: source
    path: .sdd/source
    scope: all
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte(`
schema_version: "1"
chests:
  - id: source
    path: .sdd/source
    trust:
      tier: T1
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "knowledge.index.yaml"), []byte("sources: []\n"), 0o644))

	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)
	setTreasureChestDoIndex(t, false)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	})
	assert.Contains(t, out, "missing_index")
}

func TestTreasureChestCmd_ShowsHistoricalFreshnessWarning(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: epic
base_path: .analysis
slots:
  discovery: brainstorming
  refinement: openspec-explore
  execution: sdd-ask
treasure_chests:
  - id: analysis-done
    path: .analysis/done
    scope: [discovery]
`), 0o644))
	// T2 governed but no last_reviewed → freshness=unknown
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte(`
schema_version: "1"
chests:
  - id: analysis-done
    path: .analysis/done
    trust:
      tier: T2
      reviewed_by: human
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "knowledge.index.yaml"), []byte(`
sources:
  - id: analysis-done
    tags: [discovery]
`), 0o644))

	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)
	setTreasureChestDoIndex(t, false)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	})
	assert.Contains(t, out, "unknown")       // freshness column
	assert.Contains(t, out, "last_reviewed") // warning text
}

func TestTreasureChestCmd_MissingActiveYAML(t *testing.T) {
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, filepath.Join(t.TempDir(), "nonexistent"))

	err := treasureChestCmd.RunE(treasureChestCmd, nil)
	require.Error(t, err)
}

func TestTreasureChestCmd_MissingTreasureChestsYAML_ContinuesWithWarning(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: epic
base_path: .analysis
slots:
  discovery: brainstorming
  refinement: openspec-explore
  execution: sdd-ask
treasure_chests:
  - id: source
    path: .sdd/source
    scope: all
`), 0o644))
	// No treasure-chests.yaml — must be non-fatal.

	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)
	setTreasureChestDoIndex(t, false)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	})
	assert.Contains(t, out, "CHESTS")
	assert.Contains(t, out, "source")
}

func TestTreasureChestCmd_MissingKnowledgeIndexYAML_ContinuesWithWarning(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: epic
base_path: .analysis
slots:
  discovery: brainstorming
  refinement: openspec-explore
  execution: sdd-ask
treasure_chests:
  - id: source
    path: .sdd/source
    scope: all
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte(`
schema_version: "1"
chests:
  - id: source
    path: .sdd/source
    trust:
      tier: T1
`), 0o644))
	// No knowledge.index.yaml — must be non-fatal.

	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)
	setTreasureChestDoIndex(t, false)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	})
	assert.Contains(t, out, "CHESTS")
}

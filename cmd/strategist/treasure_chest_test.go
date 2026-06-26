package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// minimalTreasureChestRoot builds a .strategist/-like tree for treasure-chest command tests.
func minimalTreasureChestRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: epic
base_path: .analysis
roles_config: roles/default.yaml
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
    title: Source
    path: .sdd/source
    trust:
      tier: T1
      reviewed_by: human
      last_reviewed: 2026-06-24
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "knowledge.index.yaml"), []byte(`
sources:
  - id: source
    path: .sdd/source
    tags: [all]
`), 0o644))

	return dir
}

// resetTreasureChestFlags saves and restores all treasure-chest command flags.
func resetTreasureChestFlags(t *testing.T) {
	t.Helper()
	origRoot := treasureChestRoot
	origIndex := treasureChestDoIndex
	origHist := treasureChestIncludeHistorical
	origFmt := treasureChestFormat
	origScope := treasureChestScope
	t.Cleanup(func() {
		treasureChestRoot = origRoot
		treasureChestDoIndex = origIndex
		treasureChestIncludeHistorical = origHist
		treasureChestFormat = origFmt
		treasureChestScope = origScope
	})
}

// --- status (read-only) tests ---

func TestTreasureChestCmd_ShowsChestsSection(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	treasureChestRoot = dir
	treasureChestDoIndex = false

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	})
	assert.Contains(t, out, "CHESTS")
	assert.Contains(t, out, "source")
	assert.Contains(t, out, "INDEX")
}

func TestTreasureChestCmd_ShowsTrustTierFromGoverned(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	treasureChestRoot = dir
	treasureChestDoIndex = false

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	})
	assert.Contains(t, out, "T1")
	assert.Contains(t, out, "fresh") // last_reviewed is set
}

func TestTreasureChestCmd_DefaultIsReadOnly(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	treasureChestRoot = dir
	treasureChestDoIndex = false

	require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))

	// Default invocation must not produce a compiled artifact.
	assert.NoFileExists(t, filepath.Join(dir, ".compiled", ".index.gz"))
}

func TestTreasureChestCmd_SuggestsIndexWhenCompiledAbsent(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	treasureChestRoot = dir
	treasureChestDoIndex = false

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
roles_config: roles/default.yaml
slots:
  discovery: brainstorming
treasure_chests:
  - id: undeclared-chest
    path: .some/path
    scope: all
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte("schema_version: \"1\"\nchests: []\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "knowledge.index.yaml"), []byte("sources: []\n"), 0o644))

	resetTreasureChestFlags(t)
	treasureChestRoot = dir
	treasureChestDoIndex = false

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
roles_config: roles/default.yaml
slots:
  discovery: brainstorming
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
	treasureChestRoot = dir
	treasureChestDoIndex = false

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
roles_config: roles/default.yaml
slots:
  discovery: brainstorming
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
	treasureChestRoot = dir
	treasureChestDoIndex = false

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	})
	assert.Contains(t, out, "unknown")       // freshness column
	assert.Contains(t, out, "last_reviewed") // warning text
}

func TestTreasureChestCmd_MissingActiveYAML(t *testing.T) {
	resetTreasureChestFlags(t)
	treasureChestRoot = filepath.Join(t.TempDir(), "nonexistent")

	err := treasureChestCmd.RunE(treasureChestCmd, nil)
	require.Error(t, err)
}

func TestTreasureChestCmd_MissingTreasureChestsYAML_ContinuesWithWarning(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: epic
base_path: .analysis
roles_config: roles/default.yaml
slots:
  discovery: brainstorming
treasure_chests:
  - id: source
    path: .sdd/source
    scope: all
`), 0o644))
	// No treasure-chests.yaml — must be non-fatal.

	resetTreasureChestFlags(t)
	treasureChestRoot = dir
	treasureChestDoIndex = false

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
roles_config: roles/default.yaml
slots:
  discovery: brainstorming
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
	treasureChestRoot = dir
	treasureChestDoIndex = false

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	})
	assert.Contains(t, out, "CHESTS")
}

// --- --index action tests ---

func TestTreasureChestCmd_IndexBuildsCompiledArtifact(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	treasureChestRoot = dir
	treasureChestDoIndex = true
	treasureChestIncludeHistorical = false

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	})
	assert.Contains(t, out, "--index complete")
	assert.FileExists(t, filepath.Join(dir, ".compiled", ".index.gz"))
}

func TestTreasureChestCmd_IndexIsIsolatedToExplicitFlag(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	treasureChestRoot = dir
	treasureChestDoIndex = false

	require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	assert.NoFileExists(t, filepath.Join(dir, ".compiled", ".index.gz"))
}

func TestTreasureChestCmd_IndexReportsHistoricalExclusionByDefault(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: epic
base_path: .analysis
roles_config: roles/default.yaml
slots:
  discovery: brainstorming
treasure_chests:
  - id: analysis-done
    path: .analysis/done
    scope: [discovery]
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte(`
schema_version: "1"
chests:
  - id: analysis-done
    path: .analysis/done
    trust:
      tier: T2
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "knowledge.index.yaml"), []byte(`
sources:
  - id: analysis-done
    tags: [discovery]
`), 0o644))

	resetTreasureChestFlags(t)
	treasureChestRoot = dir
	treasureChestDoIndex = true
	treasureChestIncludeHistorical = false

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	})
	assert.Contains(t, out, "excluded")
	assert.Contains(t, out, "--include-historical")
}

func TestTreasureChestCmd_DefaultRootFallback(t *testing.T) {
	resetTreasureChestFlags(t)
	treasureChestRoot = ""

	tmp := t.TempDir()
	origWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	err := treasureChestCmd.RunE(treasureChestCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// --- unit-level helpers ---

func TestDeriveFreshness_WithLastReviewed(t *testing.T) {
	t.Parallel()
	r := chestRow{lastReviewed: "2026-06-24"}
	assert.Equal(t, "fresh", deriveFreshness(r))
}

func TestDeriveFreshness_WithoutLastReviewed(t *testing.T) {
	t.Parallel()
	r := chestRow{}
	assert.Equal(t, "unknown", deriveFreshness(r))
}

func TestDeriveDrift_MissingGovernance(t *testing.T) {
	t.Parallel()
	r := chestRow{configured: true, governed: false, indexed: true}
	drift := deriveDrift(r)
	assert.Contains(t, drift, "missing_governance")
}

func TestDeriveDrift_MissingIndex(t *testing.T) {
	t.Parallel()
	r := chestRow{configured: true, governed: true, indexed: false}
	drift := deriveDrift(r)
	assert.Contains(t, drift, "missing_index")
}

func TestDeriveDrift_Unscoped(t *testing.T) {
	t.Parallel()
	r := chestRow{configured: false, governed: true, indexed: true}
	drift := deriveDrift(r)
	assert.Contains(t, drift, "unscoped")
}

func TestDeriveDrift_None(t *testing.T) {
	t.Parallel()
	r := chestRow{configured: true, governed: true, indexed: true}
	drift := deriveDrift(r)
	assert.Empty(t, drift)
}

func TestScopeVal_UnmarshalScalar(t *testing.T) {
	t.Parallel()
	input := []byte("scope: all\n")
	var out struct {
		Scope scopeVal `yaml:"scope"`
	}
	require.NoError(t, yaml.Unmarshal(input, &out))
	assert.Equal(t, []string{"all"}, []string(out.Scope))
}

func TestScopeVal_UnmarshalList(t *testing.T) {
	t.Parallel()
	input := []byte("scope:\n  - discovery\n  - refinement\n")
	var out struct {
		Scope scopeVal `yaml:"scope"`
	}
	require.NoError(t, yaml.Unmarshal(input, &out))
	assert.Equal(t, []string{"discovery", "refinement"}, []string(out.Scope))
}

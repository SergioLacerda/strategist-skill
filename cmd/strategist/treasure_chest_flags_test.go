package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- --index action tests ---

func TestTreasureChestCmd_IndexBuildsCompiledArtifact(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)
	setTreasureChestDoIndex(t, true)
	setTreasureChestIncludeHistorical(t, false)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	})
	assert.Contains(t, out, "--index complete")
	assert.FileExists(t, filepath.Join(dir, ".compiled", ".index.gz"))
}

func TestTreasureChestCmd_IndexIsIsolatedToExplicitFlag(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)
	setTreasureChestDoIndex(t, false)

	require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	assert.NoFileExists(t, filepath.Join(dir, ".compiled", ".index.gz"))
}

func TestTreasureChestCmd_IndexReportsHistoricalExclusionByDefault(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: epic
base_path: .analysis
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
	setTreasureChestRoot(t, dir)
	setTreasureChestDoIndex(t, true)
	setTreasureChestIncludeHistorical(t, false)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	})
	assert.Contains(t, out, "excluded")
	assert.Contains(t, out, "--include-historical")
}

func TestTreasureChestCmd_DefaultRootFallback(t *testing.T) {
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, "")

	tmp := t.TempDir()
	origWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	err := treasureChestCmd.RunE(treasureChestCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// --- --format / --scope tests (T-E1) ---

func TestTreasureChestCmd_FormatJSON_EmitsStructuredOutput(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)
	setTreasureChestFormat(t, "json")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	})

	var parsed jsonTreasureChestOutput
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	require.Len(t, parsed.Chests, 1)
	assert.Equal(t, "source", parsed.Chests[0].ID)
	assert.Equal(t, "T1", parsed.Chests[0].Trust)
	assert.NotEmpty(t, parsed.Index.Health)
}

func TestTreasureChestCmd_ShowsGradeReuseGapsColumns(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
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
    grade:
      source_grade: A
      reuse_value: high
    open_gaps: ["missing tests", "no changelog"]
`), 0o644))
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	})
	assert.Contains(t, out, "GRADE")
	assert.Contains(t, out, "REUSE")
	assert.Contains(t, out, "GAPS")
	assert.Contains(t, out, "A")
	assert.Contains(t, out, "high")
	assert.Contains(t, out, "2") // open_gaps count
}

func TestTreasureChestCmd_FormatJSON_IncludesGradeAndGaps(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte(`
schema_version: "1"
chests:
  - id: source
    title: Source
    path: .sdd/source
    trust:
      tier: T1
      reviewed_by: human
    grade:
      source_grade: B
      reuse_value: medium
    open_gaps: ["needs review"]
`), 0o644))
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)
	setTreasureChestFormat(t, "json")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	})

	var parsed jsonTreasureChestOutput
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	require.Len(t, parsed.Chests, 1)
	assert.Equal(t, "B", parsed.Chests[0].SourceGrade)
	assert.Equal(t, "medium", parsed.Chests[0].ReuseValue)
	assert.Equal(t, []string{"needs review"}, parsed.Chests[0].OpenGaps)
}

func TestTreasureChestCmd_FormatUnknown_ReturnsError(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)
	setTreasureChestFormat(t, "xml")

	err := treasureChestCmd.RunE(treasureChestCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown --format")
}

func TestTreasureChestCmd_ScopeFiltersMatchingChests(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)
	setTreasureChestScope(t, "discovery") // chest declares Scope: all, which matches any scope

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	})
	assert.Contains(t, out, "source")
}

func TestTreasureChestCmd_ScopeExcludesNonMatchingChests(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
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
    scope: refinement
`), 0o644))
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)
	setTreasureChestScope(t, "discovery")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	})
	assert.NotContains(t, out, "source")
}

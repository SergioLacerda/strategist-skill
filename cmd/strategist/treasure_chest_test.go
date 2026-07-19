package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/spf13/cobra"
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
	origRoot, err := treasureChestCmd.PersistentFlags().GetString("root")
	require.NoError(t, err)
	origIndex, err := treasureChestCmd.Flags().GetBool("index")
	require.NoError(t, err)
	origHist, err := treasureChestCmd.Flags().GetBool("include-historical")
	require.NoError(t, err)
	origFmt, err := treasureChestCmd.Flags().GetString("format")
	require.NoError(t, err)
	origScope, err := treasureChestCmd.Flags().GetString("scope")
	require.NoError(t, err)
	t.Cleanup(func() {
		setTreasureChestRoot(t, origRoot)
		setTreasureChestDoIndex(t, origIndex)
		setTreasureChestIncludeHistorical(t, origHist)
		setTreasureChestFormat(t, origFmt)
		setTreasureChestScope(t, origScope)
	})
	setTreasureChestRoot(t, "")
	setTreasureChestDoIndex(t, false)
	setTreasureChestIncludeHistorical(t, false)
	setTreasureChestFormat(t, "table")
	setTreasureChestScope(t, "")
	setCmdFlag(t, treasureChestScanCmd, "dry-run", "false")
}

func setTreasureChestRoot(t *testing.T, value string) {
	t.Helper()
	require.NoError(t, treasureChestCmd.PersistentFlags().Set("root", value))
}

func setTreasureChestDoIndex(t *testing.T, value bool) {
	t.Helper()
	require.NoError(t, treasureChestCmd.Flags().Set("index", fmt.Sprint(value)))
}

func setTreasureChestIncludeHistorical(t *testing.T, value bool) {
	t.Helper()
	require.NoError(t, treasureChestCmd.Flags().Set("include-historical", fmt.Sprint(value)))
}

func setTreasureChestFormat(t *testing.T, value string) {
	t.Helper()
	require.NoError(t, treasureChestCmd.Flags().Set("format", value))
}

func setTreasureChestScope(t *testing.T, value string) {
	t.Helper()
	require.NoError(t, treasureChestCmd.Flags().Set("scope", value))
}

func setCmdFlag(t *testing.T, cmd *cobra.Command, name, value string) {
	t.Helper()
	require.NoError(t, cmd.Flags().Set(name, value))
}

func cmdFlagString(t *testing.T, cmd *cobra.Command, name string) string {
	t.Helper()
	value, err := cmd.Flags().GetString(name)
	require.NoError(t, err)
	return value
}

func cmdFlagBool(t *testing.T, cmd *cobra.Command, name string) bool {
	t.Helper()
	value, err := cmd.Flags().GetBool(name)
	require.NoError(t, err)
	return value
}

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

func TestFilterRowsByScope_EmptyValueIsNoop(t *testing.T) {
	t.Parallel()
	rows := []treasure.StatusRow{{ID: "a", Scope: []string{"discovery"}}}
	assert.Equal(t, rows, treasure.FilterRowsByScope(rows, ""))
}

func TestFilterRowsByScope_MatchesAllScope(t *testing.T) {
	t.Parallel()
	rows := []treasure.StatusRow{{ID: "a", Scope: []string{"all"}}}
	assert.Len(t, treasure.FilterRowsByScope(rows, "execution"), 1)
}

func TestFilterRowsByScope_ExcludesUnscopedRows(t *testing.T) {
	t.Parallel()
	rows := []treasure.StatusRow{{ID: "a", Scope: nil}}
	assert.Empty(t, treasure.FilterRowsByScope(rows, "discovery"))
}

// --- unit-level helpers ---

func TestDeriveFreshness_WithLastReviewed(t *testing.T) {
	t.Parallel()
	r := treasure.StatusRow{LastReviewed: "2026-06-24"}
	assert.Equal(t, "fresh", treasure.DeriveFreshness(r))
}

func TestDeriveFreshness_WithoutLastReviewed(t *testing.T) {
	t.Parallel()
	r := treasure.StatusRow{}
	assert.Equal(t, "unknown", treasure.DeriveFreshness(r))
}

func TestDeriveDrift_MissingGovernance(t *testing.T) {
	t.Parallel()
	r := treasure.StatusRow{Configured: true, Governed: false, Indexed: true}
	drift := treasure.DeriveDrift(r)
	assert.Contains(t, drift, "missing_governance")
}

func TestDeriveDrift_MissingIndex(t *testing.T) {
	t.Parallel()
	r := treasure.StatusRow{Configured: true, Governed: true, Indexed: false}
	drift := treasure.DeriveDrift(r)
	assert.Contains(t, drift, "missing_index")
}

func TestDeriveDrift_Unscoped(t *testing.T) {
	t.Parallel()
	r := treasure.StatusRow{Configured: false, Governed: true, Indexed: true}
	drift := treasure.DeriveDrift(r)
	assert.Contains(t, drift, "unscoped")
}

func TestDeriveDrift_None(t *testing.T) {
	t.Parallel()
	r := treasure.StatusRow{Configured: true, Governed: true, Indexed: true}
	drift := treasure.DeriveDrift(r)
	assert.Empty(t, drift)
}

func TestScopeVal_UnmarshalScalar(t *testing.T) {
	t.Parallel()
	input := []byte("scope: all\n")
	var out struct {
		Scope treasure.Scope `yaml:"scope"`
	}
	require.NoError(t, yaml.Unmarshal(input, &out))
	assert.Equal(t, []string{"all"}, []string(out.Scope))
}

func TestScopeVal_UnmarshalList(t *testing.T) {
	t.Parallel()
	input := []byte("scope:\n  - discovery\n  - refinement\n")
	var out struct {
		Scope treasure.Scope `yaml:"scope"`
	}
	require.NoError(t, yaml.Unmarshal(input, &out))
	assert.Equal(t, []string{"discovery", "refinement"}, []string(out.Scope))
}

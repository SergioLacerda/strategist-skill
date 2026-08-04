package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setEvalHarvestFlags(t *testing.T, root string, all bool, include string) {
	t.Helper()
	require.NoError(t, evalHarvestCmd.Flags().Set(flagRoot, root))
	require.NoError(t, evalHarvestCmd.Flags().Set("all", boolFlagString(all)))
	require.NoError(t, evalHarvestCmd.Flags().Set("include", include))
}

func boolFlagString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func resetEvalHarvestFlags(t *testing.T) {
	t.Helper()
	setEvalHarvestFlags(t, "", false, "")
}

// writeMissionAnalysis writes a minimal analysis.md into a mission
// directory, alongside whatever writeMissionTasks (treasure_chest_scan_test.go)
// already wrote for tasks.md — harvest's default artifact is analysis.md.
func writeMissionAnalysis(t *testing.T, basePath, category, missionID, content string) {
	t.Helper()
	dir := filepath.Join(basePath, category, missionID)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "analysis.md"), []byte(content), 0o644))
}

func TestMissionDir_PrefersRefinedOverDone(t *testing.T) {
	base := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(base, "refined", "m-1"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(base, "done", "m-1"), 0o755))

	dir, err := missionDir(base, "m-1")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(base, "refined", "m-1"), dir)
}

func TestMissionDir_FallsBackToDone(t *testing.T) {
	base := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(base, "done", "m-2"), 0o755))

	dir, err := missionDir(base, "m-2")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(base, "done", "m-2"), dir)
}

func TestMissionDir_NotFound(t *testing.T) {
	base := t.TempDir()
	_, err := missionDir(base, "missing-mission")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing-mission")
}

func TestParseHarvestInclude_Empty(t *testing.T) {
	out, err := parseHarvestInclude("")
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestParseHarvestInclude_ValidList(t *testing.T) {
	out, err := parseHarvestInclude("design, tasks,adr")
	require.NoError(t, err)
	assert.Equal(t, []string{"design", "tasks", "adr"}, out)
}

func TestParseHarvestInclude_UnknownValue(t *testing.T) {
	_, err := parseHarvestInclude("bogus")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus")
}

func TestSelectHarvestMissionIDs_SingleMission(t *testing.T) {
	ids, warnings, err := selectHarvestMissionIDs([]string{"m-1"}, evalHarvestOptions{}, "")
	require.NoError(t, err)
	assert.Equal(t, []string{"m-1"}, ids)
	assert.Nil(t, warnings)
}

func TestSelectHarvestMissionIDs_RequiresExactlyOneWithoutAll(t *testing.T) {
	_, _, err := selectHarvestMissionIDs(nil, evalHarvestOptions{}, "")
	require.Error(t, err)

	_, _, err = selectHarvestMissionIDs([]string{"a", "b"}, evalHarvestOptions{}, "")
	require.Error(t, err)
}

func TestSelectHarvestMissionIDs_AllRejectsMissionID(t *testing.T) {
	_, _, err := selectHarvestMissionIDs([]string{"m-1"}, evalHarvestOptions{All: true}, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestSelectHarvestMissionIDs_AllScansRefinedAndDone(t *testing.T) {
	_, base := scanTestRoot(t)
	writeMissionTasks(t, base, "refined", "m-refined", "## Task 1 — Example\n")
	writeMissionTasks(t, base, "done", "m-done", "## Task 1 — Example\n")

	ids, warnings, err := selectHarvestMissionIDs(nil, evalHarvestOptions{All: true}, base)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"m-refined", "m-done"}, ids)
	assert.Empty(t, warnings)
}

// TestSelectHarvestMissionIDs_AllSkipsInconsistentMission reproduces the real
// failure behind .analysis/archived/20260804-treasure-scan-sq-block-bug-adr.md:
// a side_quests_approved: block using standard 2-space-indented list items
// (as opposed to the 0-indent style used elsewhere in this test file) trips
// scan_parse.go's normalizeLegacySideQuestFields, which strips the leading
// "- " from the indented first item, corrupting the YAML. --all must skip
// that mission with a warning instead of aborting the whole run.
func TestSelectHarvestMissionIDs_AllSkipsInconsistentMission(t *testing.T) {
	_, base := scanTestRoot(t)
	writeMissionTasks(t, base, "done", "good-mission", "## Task 1 — Example\n")
	writeMissionTasks(t, base, "done", "bad-mission",
		"side_quests_approved:\n  - id: SQ-001\n    description: indented list item\n    status: sq_backlog\n## Task 1 — Example\n")

	ids, warnings, err := selectHarvestMissionIDs(nil, evalHarvestOptions{All: true}, base)

	require.NoError(t, err)
	assert.Equal(t, []string{"good-mission"}, ids)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0].Path, "bad-mission")
}

func TestHarvestMission_CopiesDefaultAnalysisFile(t *testing.T) {
	_, base := scanTestRoot(t)
	writeMissionTasks(t, base, "refined", "m-1", "## Task 1 — Example\n")
	writeMissionAnalysis(t, base, "refined", "m-1", "# Analysis\n\nreal content\n")

	dest := t.TempDir()
	n, err := harvestMission(base, dest, "m-1", nil)
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	got, err := os.ReadFile(filepath.Join(dest, "m-1", "analysis.md"))
	require.NoError(t, err)
	assert.Equal(t, "# Analysis\n\nreal content\n", string(got))
}

func TestHarvestMission_IncludeTasksAndAdr(t *testing.T) {
	_, base := scanTestRoot(t)
	writeMissionTasks(t, base, "refined", "m-1", "## Task 1 — Example\n")
	writeMissionAnalysis(t, base, "refined", "m-1", "# Analysis\n")
	require.NoError(t, os.MkdirAll(filepath.Join(base, "archived"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(base, "archived", "m-1-adr.md"), []byte("# ADR\n"), 0o644))

	dest := t.TempDir()
	n, err := harvestMission(base, dest, "m-1", []string{"tasks", "adr"})
	require.NoError(t, err)
	assert.Equal(t, 3, n)

	assert.FileExists(t, filepath.Join(dest, "m-1", "analysis.md"))
	assert.FileExists(t, filepath.Join(dest, "m-1", "tasks.md"))
	got, err := os.ReadFile(filepath.Join(dest, "m-1", "adr.md"))
	require.NoError(t, err)
	assert.Equal(t, "# ADR\n", string(got))
}

func TestHarvestMission_MissingMissionErrors(t *testing.T) {
	base := t.TempDir()
	dest := t.TempDir()
	_, err := harvestMission(base, dest, "nope", nil)
	require.Error(t, err)
}

func TestHarvestMission_ReharvestOverwrites(t *testing.T) {
	_, base := scanTestRoot(t)
	writeMissionTasks(t, base, "refined", "m-1", "## Task 1 — Example\n")
	writeMissionAnalysis(t, base, "refined", "m-1", "first version\n")

	dest := t.TempDir()
	_, err := harvestMission(base, dest, "m-1", nil)
	require.NoError(t, err)

	writeMissionAnalysis(t, base, "refined", "m-1", "second version\n")
	_, err = harvestMission(base, dest, "m-1", nil)
	require.NoError(t, err)

	got, err := os.ReadFile(filepath.Join(dest, "m-1", "analysis.md"))
	require.NoError(t, err)
	assert.Equal(t, "second version\n", string(got))
}

func TestEvalHarvestCmd_EndToEnd(t *testing.T) {
	dir, base := scanTestRoot(t)
	writeMissionTasks(t, base, "refined", "20260804-example", "## Task 1 — Example\n")
	writeMissionAnalysis(t, base, "refined", "20260804-example", "# Analysis\n\nreal fixture content\n")

	resetEvalHarvestFlags(t)
	setEvalHarvestFlags(t, dir, false, "")
	t.Cleanup(func() { resetEvalHarvestFlags(t) })

	out := captureStdout(t, func() {
		require.NoError(t, evalHarvestCmd.RunE(evalHarvestCmd, []string{"20260804-example"}))
	})
	assert.Contains(t, out, "1 mission(s), 1 fixture file(s) written")

	projectRoot := filepath.Dir(dir)
	got, err := os.ReadFile(filepath.Join(projectRoot, "tests", "evals", "regression", "20260804-example", "analysis.md"))
	require.NoError(t, err)
	assert.Equal(t, "# Analysis\n\nreal fixture content\n", string(got))
}

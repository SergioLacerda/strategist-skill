package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/spf13/cobra"
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
// directory, alongside whatever writeMissionTasks already wrote for
// tasks.md — harvest's default artifact is analysis.md.
func writeMissionAnalysis(t *testing.T, basePath, category, missionID, content string) {
	t.Helper()
	dir := filepath.Join(basePath, category, missionID)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "analysis.md"), []byte(content), 0o644))
}

// scanTestRoot builds a project tree with a .strategist/ root and an .analysis/
// base_path, matching resolveDojoRoots' expected layout (base_path resolved
// relative to the .strategist root's parent directory). Duplicated from
// internal/treasurecli's own treasure_chest_scan_test.go — Go test helpers
// aren't shareable across package boundaries, and this fixture builder isn't
// worth promoting to an exported package just for two callers.
func scanTestRoot(t *testing.T) (strategistDir, basePath string) {
	t.Helper()
	root := t.TempDir()
	strategistDir = filepath.Join(root, ".strategist")
	basePath = filepath.Join(root, ".analysis")
	require.NoError(t, os.MkdirAll(strategistDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(strategistDir, "active.yaml"), []byte(`
mode: epic
base_path: .analysis
slots:
  discovery: brainstorming
  refinement: openspec-explore
  execution: sniper
`), 0o644))
	return strategistDir, basePath
}

// writeMissionTasks writes a minimal tasks.md into a mission directory.
// Duplicated from internal/treasurecli's own treasure_chest_scan_test.go —
// see scanTestRoot's comment above.
func writeMissionTasks(t *testing.T, basePath, category, missionID, content string) {
	t.Helper()
	dir := filepath.Join(basePath, category, missionID)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644))
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

func TestHarvestIncludeSource_ReportCase(t *testing.T) {
	src, dest := harvestIncludeSource("/base", "/src", "/dest", "m-1", "report")
	assert.Equal(t, filepath.Join("/base", "archived", "m-1-report.md"), src)
	assert.Equal(t, filepath.Join("/dest", "report.md"), dest)
}

func TestHarvestMission_DestMkdirAllFailsWhenParentIsFile(t *testing.T) {
	_, base := scanTestRoot(t)
	writeMissionTasks(t, base, "refined", "m-1", "## Task 1 — Example\n")
	writeMissionAnalysis(t, base, "refined", "m-1", "# Analysis\n")

	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))
	dest := filepath.Join(blocker, "dest")

	_, err := harvestMission(base, dest, "m-1", nil)
	require.Error(t, err)
}

func TestHarvestMission_MissingAnalysisFilePropagatesCopyError(t *testing.T) {
	_, base := scanTestRoot(t)
	writeMissionTasks(t, base, "refined", "m-1", "## Task 1 — Example\n")
	// analysis.md deliberately not written — missionDir succeeds (the
	// mission directory exists via writeMissionTasks) but copying the
	// default analysis.md must fail.

	dest := t.TempDir()
	_, err := harvestMission(base, dest, "m-1", nil)
	require.Error(t, err)
}

func TestHarvestMission_MissingIncludeFilePropagatesCopyError(t *testing.T) {
	_, base := scanTestRoot(t)
	writeMissionTasks(t, base, "refined", "m-1", "## Task 1 — Example\n")
	writeMissionAnalysis(t, base, "refined", "m-1", "# Analysis\n")
	// "design" is a valid include type but design.md is never written —
	// the analysis.md copy succeeds first, then the include-loop copy
	// fails.

	dest := t.TempDir()
	_, err := harvestMission(base, dest, "m-1", []string{"design"})
	require.Error(t, err)
}

func TestCopyHarvestFile_OpenError(t *testing.T) {
	dir := t.TempDir()
	err := copyHarvestFile(filepath.Join(dir, "does-not-exist"), filepath.Join(dir, "out"))
	require.ErrorContains(t, err, "read")
}

func TestCopyHarvestFile_CreateErrorWhenDestParentMissing(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	require.NoError(t, os.WriteFile(src, []byte("content"), 0o644))

	err := copyHarvestFile(src, filepath.Join(dir, "no-such-dir", "out"))
	require.ErrorContains(t, err, "write")
}

func TestSelectAllHarvestMissionIDs_ScanErrorPropagates(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ENOTDIR scan semantics differ on Windows")
	}
	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))

	_, _, err := selectAllHarvestMissionIDs(blocker)
	require.ErrorContains(t, err, "eval harvest")
}

func TestPrintEvalHarvestWarnings_PrintsEachWarning(t *testing.T) {
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetErr(&buf)

	printEvalHarvestWarnings(cmd, []treasure.ScanWarning{{Path: "refined/bad-mission", Err: errors.New("unparseable")}})
	assert.Contains(t, buf.String(), "skipped inconsistent mission file")
}

func TestParseHarvestInclude_SkipsBlankSegments(t *testing.T) {
	out, err := parseHarvestInclude("design,,tasks")
	require.NoError(t, err)
	assert.Equal(t, []string{"design", "tasks"}, out)
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

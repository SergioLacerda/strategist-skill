package eval

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeMissionAnalysis writes a minimal analysis.md into a mission
// directory, alongside whatever writeMissionTasks already wrote for
// tasks.md — harvest's default artifact is analysis.md.
func writeMissionAnalysis(t *testing.T, basePath, category, missionID, content string) {
	t.Helper()
	dir := filepath.Join(basePath, category, missionID)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "analysis.md"), []byte(content), 0o644))
}

// writeMissionTasks writes a minimal tasks.md into a mission directory.
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

func TestHarvestMission_CopiesDefaultAnalysisFile(t *testing.T) {
	base := t.TempDir()
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
	base := t.TempDir()
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
	base := t.TempDir()
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
	base := t.TempDir()
	writeMissionTasks(t, base, "refined", "m-1", "## Task 1 — Example\n")
	writeMissionAnalysis(t, base, "refined", "m-1", "# Analysis\n")

	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))
	dest := filepath.Join(blocker, "dest")

	_, err := harvestMission(base, dest, "m-1", nil)
	require.Error(t, err)
}

func TestHarvestMission_MissingAnalysisFilePropagatesCopyError(t *testing.T) {
	base := t.TempDir()
	writeMissionTasks(t, base, "refined", "m-1", "## Task 1 — Example\n")
	// analysis.md deliberately not written — missionDir succeeds (the
	// mission directory exists via writeMissionTasks) but copying the
	// default analysis.md must fail.

	dest := t.TempDir()
	_, err := harvestMission(base, dest, "m-1", nil)
	require.Error(t, err)
}

func TestHarvestMission_MissingIncludeFilePropagatesCopyError(t *testing.T) {
	base := t.TempDir()
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

func TestCopyHarvestFile_CopyErrorWhenSrcIsDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("reading a directory as a file has different semantics on Windows")
	}
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src-dir")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))

	err := copyHarvestFile(srcDir, filepath.Join(dir, "out"))
	require.ErrorContains(t, err, "copy")
}

package treasure

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanMissionsInDir_ReadDirPermissionError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	sub := filepath.Join(dir, "refined")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	require.NoError(t, os.Chmod(sub, 0o000))
	t.Cleanup(func() { _ = os.Chmod(sub, 0o755) })

	_, err := ScanMissionsInDir(sub)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read")
}

func TestRunScanPipeline_HappyPath(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	basePath := filepath.Join(root, ".analysis")
	missionDir := filepath.Join(basePath, "refined", "mission-a")
	require.NoError(t, os.MkdirAll(missionDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(missionDir, "tasks.md"),
		[]byte("## Task 1 — Extract governance sync\n\nside_quests_approved:\n\n```yaml\n- id: SQ-001\n  description: follow-up\n  strategy: execute_later\n  estimated_impact: low\n  dependencies: []\n  status: sq_pending\n```\n"),
		0o644))

	result, err := RunScanPipeline(root, basePath)
	require.NoError(t, err)
	require.Len(t, result.Missions, 1)
	assert.Equal(t, "mission-a", result.Missions[0].MissionID)
	assert.Empty(t, result.Warnings)

	assert.DirExists(t, filepath.Join(root, "treasure", "clusters"))
	assert.DirExists(t, filepath.Join(root, "treasure", "gaps"))
	assert.FileExists(t, filepath.Join(root, "treasure", "gaps", result.Gaps[0].ID+".md"))
}

func TestRunScanPipeline_ScanError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	root := t.TempDir()
	basePath := filepath.Join(root, ".analysis")
	refined := filepath.Join(basePath, "refined")
	require.NoError(t, os.MkdirAll(refined, 0o755))
	require.NoError(t, os.Chmod(refined, 0o000))
	t.Cleanup(func() { _ = os.Chmod(refined, 0o755) })

	_, err := RunScanPipeline(root, basePath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read")
}

func TestScanMissionsInDir_PropagatesParseError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	missionDir := filepath.Join(dir, "mission-a")
	require.NoError(t, os.MkdirAll(missionDir, 0o755))
	// side_quests_approved: block with invalid YAML underneath.
	require.NoError(t, os.WriteFile(filepath.Join(missionDir, "tasks.md"),
		[]byte("side_quests_approved:\n  : not: valID:\n"), 0o644))

	_, err := ScanMissionsInDir(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse side_quests_approved")
}

// TestFilterRowsByScope_MixedRoster mirrors the roster shape a real Ranger/
// Archivist scope filter operates on: some chests scoped to a single slot,
// one scoped to "all", one scoped to an unrelated slot — and asserts the
// filtered result is exactly the expected set, not merely non-empty
// (status_test.go's existing FilterRowsByScope tests each cover one
// single-row edge case in isolation; this one exercises them together).
func TestFilterRowsByScope_MixedRoster(t *testing.T) {
	t.Parallel()
	rows := []StatusRow{
		{ID: "runbooks", Scope: []string{"all"}},
		{ID: "discovery-notes", Scope: []string{"discovery"}},
		{ID: "execution-only", Scope: []string{"execution"}},
		{ID: "unscoped", Scope: nil},
	}

	got := FilterRowsByScope(rows, "discovery")

	gotIDs := make([]string, 0, len(got))
	for _, r := range got {
		gotIDs = append(gotIDs, r.ID)
	}
	assert.ElementsMatch(t, []string{"runbooks", "discovery-notes"}, gotIDs)
}

func TestScanWarning_Error(t *testing.T) {
	t.Parallel()
	w := ScanWarning{Path: "refined/m1", Err: errors.New("boom")}
	assert.Equal(t, "refined/m1: boom", w.Error())

	wNoPath := ScanWarning{Err: errors.New("boom")}
	assert.Equal(t, "boom", wNoPath.Error())
}

func TestScanMissions_ScansRefinedAndDone(t *testing.T) {
	t.Parallel()
	basePath := t.TempDir()
	for _, sub := range []string{"refined", "done"} {
		missionDir := filepath.Join(basePath, sub, "mission-"+sub)
		require.NoError(t, os.MkdirAll(missionDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(missionDir, "tasks.md"), []byte("## Task 1\n"), 0o644))
	}

	got, err := ScanMissions(basePath)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "mission-done", got[0].MissionID)
	assert.Equal(t, "mission-refined", got[1].MissionID)
}

func TestScanMissions_MissingBaseDirsIsEmpty(t *testing.T) {
	t.Parallel()
	got, err := ScanMissions(t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestScanMissions_ErrorPropagates(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	basePath := t.TempDir()
	refined := filepath.Join(basePath, "refined")
	require.NoError(t, os.MkdirAll(refined, 0o755))
	require.NoError(t, os.Chmod(refined, 0o000))
	t.Cleanup(func() { _ = os.Chmod(refined, 0o755) })

	_, err := ScanMissions(basePath)
	require.Error(t, err)
}

func TestScanMissionsTolerant_SortsMultipleMissions(t *testing.T) {
	t.Parallel()
	basePath := t.TempDir()
	for _, id := range []string{"mission-b", "mission-a"} {
		missionDir := filepath.Join(basePath, "refined", id)
		require.NoError(t, os.MkdirAll(missionDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(missionDir, "tasks.md"), []byte("## Task 1\n"), 0o644))
	}

	got, warnings, err := ScanMissionsTolerant(basePath)
	require.NoError(t, err)
	assert.Empty(t, warnings)
	require.Len(t, got, 2)
	assert.Equal(t, "mission-a", got[0].MissionID)
	assert.Equal(t, "mission-b", got[1].MissionID)
}

func TestScanMissionsInDirTolerant_SkipsMalformedMissionAsWarning(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	goodDir := filepath.Join(dir, "mission-good")
	require.NoError(t, os.MkdirAll(goodDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(goodDir, "tasks.md"), []byte("## Task 1\n"), 0o644))

	badDir := filepath.Join(dir, "mission-bad")
	require.NoError(t, os.MkdirAll(badDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(badDir, "tasks.md"),
		[]byte("side_quests_approved:\n  : not: valID:\n"), 0o644))

	missions, warnings, err := ScanMissionsInDirTolerant(dir)
	require.NoError(t, err)
	require.Len(t, missions, 1)
	assert.Equal(t, "mission-good", missions[0].MissionID)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0].Path, "mission-bad")
}

func TestScanMissionDirEntry_SkipsNonDirEntries(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "stray-file.md"), nil, 0o644))

	got, err := ScanMissionsInDir(dir)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestScanMissionDirEntry_NoTasksFileIsNotError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "empty-mission"), 0o755))

	got, err := ScanMissionsInDir(dir)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestRunScanPipeline_WriteScanOutputsErrorPropagates(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	root := t.TempDir()
	basePath := filepath.Join(root, ".analysis")
	missionDir := filepath.Join(basePath, "refined", "mission-a")
	require.NoError(t, os.MkdirAll(missionDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(missionDir, "tasks.md"), []byte("## Task 1\n"), 0o644))

	// Make "treasure" (WriteScanOutputs' parent dir) unwritable so it can't
	// create the clusters/gaps subdirectories.
	require.NoError(t, os.MkdirAll(filepath.Join(root, "treasure"), 0o755))
	require.NoError(t, os.Chmod(filepath.Join(root, "treasure"), 0o000))
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(root, "treasure"), 0o755) })

	_, err := RunScanPipeline(root, basePath)
	require.Error(t, err)
}

func TestScanMissionWarning_BuildsTasksPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	missionDir := filepath.Join(dir, "mission-a")
	require.NoError(t, os.MkdirAll(missionDir, 0o755))
	dirEntries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, dirEntries, 1)

	w := scanMissionWarning(dir, dirEntries[0], errors.New("boom"))
	assert.Equal(t, filepath.Join(dir, "mission-a", "tasks.md"), w.Path)
	assert.ErrorContains(t, w.Err, "boom")
}

package treasure

import (
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

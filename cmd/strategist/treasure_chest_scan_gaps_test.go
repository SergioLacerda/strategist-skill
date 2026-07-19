package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Whitebox tests targeting uncovered branches in treasure_chest_scan.go: command-level
// error paths (resolveStrategistRoot/resolveDojoRoots/treasure.ScanMissions/treasure.WriteScanOutputs
// failures), permission-based I/O failures in the writer helpers, and pure-function edge
// cases (multi-treasure.Cluster sort, duplicate treasure.Gap citation, id truncation) not exercised by the
// happy-path tests in treasure_chest_scan_test.go.

// --- runTreasureChestScan: resolveStrategistRoot / resolveDojoRoots errors ---

func TestTreasureChestScan_ResolveRootError(t *testing.T) {
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, "")

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	require.NoError(t, os.Chdir(t.TempDir()))

	err = treasureChestScanCmd.RunE(treasureChestScanCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest scan")
}

func TestTreasureChestScan_ResolveDojoRootsError(t *testing.T) {
	dir, _ := scanTestRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(": not: valID: yaml:\n"), 0o644))
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	err := treasureChestScanCmd.RunE(treasureChestScanCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest scan")
}

// --- runTreasureChestScan: treasure.ScanMissions error (unreadable refined/ dir) ---

func TestTreasureChestScan_ScanMissionsError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir, basePath := scanTestRoot(t)
	refinedDir := filepath.Join(basePath, "refined")
	require.NoError(t, os.MkdirAll(refinedDir, 0o755))
	require.NoError(t, os.Chmod(refinedDir, 0o000))
	t.Cleanup(func() { _ = os.Chmod(refinedDir, 0o755) })
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	err := treasureChestScanCmd.RunE(treasureChestScanCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest scan")
}

// --- runTreasureChestScan: treasure.WriteScanOutputs error (clustersDir parent unwritable) ---

func TestTreasureChestScan_WriteScanOutputsError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir, basePath := scanTestRoot(t)
	writeMissionTasks(t, basePath, "refined", "mission-a", "side_quests_approved:\n\n- id: `SQ-101`\n  description: Pending item.\n  status: `sq_pending`\n")
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	err := treasureChestScanCmd.RunE(treasureChestScanCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest scan")
}

// --- printScanDryRun: loop bodies with actual clusters and gaps ---

func TestTreasureChestScan_DryRunListsClustersAndGaps(t *testing.T) {
	dir, basePath := scanTestRoot(t)
	writeMissionTasks(t, basePath, "refined", "mission-a", "side_quests_approved:\n\n- id: `SQ-101`\n  description: Improve widget caching layer for faster loads.\n  status: `sq_pending`\n\n## Suggested Validation\n")
	writeMissionTasks(t, basePath, "refined", "mission-b", "## Task 1 — Improve widget rendering\n\nside_quests_approved:\n\n- id: `SQ-102`\n  description: Improve widget caching consistency.\n  status: `sq_pending`\n\n## Suggested Validation\n")
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestScanCmd, "dry-run", "true")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestScanCmd.RunE(treasureChestScanCmd, nil))
	})
	assert.Contains(t, out, "cluster: ")
	assert.Contains(t, out, "gap: ")
}

// --- ScanMissionsInDir: readdir permission error ---

func TestScanMissionsInDir_ReadDirPermissionError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	sub := filepath.Join(dir, "refined")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	require.NoError(t, os.Chmod(sub, 0o000))
	t.Cleanup(func() { _ = os.Chmod(sub, 0o755) })

	_, err := treasure.ScanMissionsInDir(sub)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read")
}

// --- ScanMissionsInDir / ParseMissionTasks: propagated parse error ---

func TestScanMissionsInDir_PropagatesParseError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	missionDir := filepath.Join(dir, "mission-a")
	require.NoError(t, os.MkdirAll(missionDir, 0o755))
	// side_quests_approved: block with invalid YAML underneath.
	require.NoError(t, os.WriteFile(filepath.Join(missionDir, "tasks.md"),
		[]byte("side_quests_approved:\n  : not: valID:\n"), 0o644))

	_, err := treasure.ScanMissionsInDir(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse side_quests_approved")
}

func TestParseMissionTasks_InvalidYAML(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "tasks.md")
	require.NoError(t, os.WriteFile(path, []byte("side_quests_approved:\n  : not: valID:\n"), 0o644))

	_, err := treasure.ParseMissionTasks("mission-x", path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse side_quests_approved")
}

func TestParseMissionTasks_FencedSideQuestsYAML(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "tasks.md")
	require.NoError(t, os.WriteFile(path, []byte(`
## Archivist-To-Sniper Handoff Fields

side_quests_approved:

`+"```yaml"+`
- id: SQ-001
  description: Keep fenced side quests parseable.
  strategy: execute_later
  estimated_impact: low
  dependencies: []
  status: sq_pending
`+"```"+`

acceptance_checks:

`+"```yaml"+`
- "unrelated fenced YAML remains outside side_quests_approved"
`+"```"+`
`), 0o644))

	mission, err := treasure.ParseMissionTasks("mission-x", path)
	require.NoError(t, err)
	require.Len(t, mission.SQs, 1)
	assert.Equal(t, "SQ-001", mission.SQs[0].ID)
	assert.Equal(t, "sq_pending", mission.SQs[0].Status)
}

func TestParseMissionTasks_FencedSideQuestsYAMLWithBacktickScalars(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "tasks.md")
	require.NoError(t, os.WriteFile(path, []byte("side_quests_approved:\n\n```yml\n- id: `SQ-002`\n  description: Backtick scalars still parse.\n  dependencies: [`SQ-001`]\n  status: `sq_pending`\n```\n"), 0o644))

	mission, err := treasure.ParseMissionTasks("mission-x", path)
	require.NoError(t, err)
	require.Len(t, mission.SQs, 1)
	assert.Equal(t, "SQ-002", mission.SQs[0].ID)
	assert.Equal(t, []string{"SQ-001"}, mission.SQs[0].Dependencies)
	assert.Equal(t, "sq_pending", mission.SQs[0].Status)
}

func TestParseMissionTasks_LegacySideQuestFieldBullets(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "tasks.md")
	require.NoError(t, os.WriteFile(path, []byte("side_quests_approved:\n\n- id: `SQ-003`\n  - description: Legacy field bullets still parse.\n  - strategy: `execute_later`\n  - estimated_impact: `medium`\n  - dependencies: none\n  - status: `sq_backlog`\n"), 0o644))

	mission, err := treasure.ParseMissionTasks("mission-x", path)
	require.NoError(t, err)
	require.Len(t, mission.SQs, 1)
	assert.Equal(t, "SQ-003", mission.SQs[0].ID)
	assert.Equal(t, "Legacy field bullets still parse.", mission.SQs[0].Description)
	assert.Equal(t, "execute_later", mission.SQs[0].Strategy)
	assert.Empty(t, mission.SQs[0].Dependencies)
	assert.Equal(t, "sq_backlog", mission.SQs[0].Status)
}

func TestParseMissionTasks_MissingFile(t *testing.T) {
	t.Parallel()
	_, err := treasure.ParseMissionTasks("mission-x", filepath.Join(t.TempDir(), "nope.md"))
	require.Error(t, err)
}

// --- RegenerateDir ---

func TestRegenerateDir_MkdirAllError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	parent := t.TempDir()
	require.NoError(t, os.Chmod(parent, 0o555))
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })

	err := treasure.RegenerateDir(filepath.Join(parent, "child"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create")
}

func TestRegenerateDir_RemoveAllError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	parent := t.TempDir()
	target := filepath.Join(parent, "child")
	require.NoError(t, os.MkdirAll(target, 0o755))
	require.NoError(t, os.Chmod(parent, 0o555))
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })

	err := treasure.RegenerateDir(target)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remove")
}

func TestRegenerateDir_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, "child")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "stray"), []byte("x"), 0o644)) // unrelated file, unaffected
	require.NoError(t, treasure.RegenerateDir(target))
	info, err := os.Stat(target)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// --- treasure.WriteClusterFile / treasure.WriteGapFile: write errors ---

func TestWriteClusterFile_WriteError(t *testing.T) {
	t.Parallel()
	// Directory does not exist -> os.WriteFile fails.
	err := treasure.WriteClusterFile(filepath.Join(t.TempDir(), "nonexistent"), treasure.Cluster{ID: "cluster-x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write")
}

func TestWriteGapFile_WriteError(t *testing.T) {
	t.Parallel()
	err := treasure.WriteGapFile(filepath.Join(t.TempDir(), "nonexistent"), treasure.Gap{ID: "sq-001"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write")
}

func TestWriteGapFile_WithDependencies(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, treasure.WriteGapFile(dir, treasure.Gap{ID: "sq-001", Dependencies: []string{"sq-000"}, Status: "sq_pending"}))
	raw, err := os.ReadFile(filepath.Join(dir, "sq-001.md"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "dependencies: [sq-000]")
}

// --- treasure.WriteScanOutputs: gapsDir regeneration failure (second RegenerateDir call) ---

func TestWriteScanOutputs_GapsDirError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	parent := t.TempDir()
	clustersDir := filepath.Join(parent, "clusters")
	gapsDir := filepath.Join(parent, "locked", "gaps")
	require.NoError(t, os.MkdirAll(filepath.Join(parent, "locked"), 0o555))
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(parent, "locked"), 0o755) })

	err := treasure.WriteScanOutputs(clustersDir, nil, gapsDir, []treasure.Gap{{ID: "sq-001", Status: "sq_pending"}})
	require.Error(t, err)
}

func TestWriteScanOutputs_ClusterWriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	parent := t.TempDir()
	clustersDir := filepath.Join(parent, "locked", "clusters")
	require.NoError(t, os.MkdirAll(filepath.Join(parent, "locked"), 0o555))
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(parent, "locked"), 0o755) })

	err := treasure.WriteScanOutputs(clustersDir, []treasure.Cluster{{ID: "cluster-x"}}, filepath.Join(parent, "gaps"), nil)
	require.Error(t, err)
}

// --- treasure.BuildClusters: multiple clusters (exercises the sort.Slice comparator) ---

func TestBuildClusters_SortsMultipleClusters(t *testing.T) {
	t.Parallel()
	missions := []treasure.ScannedMission{
		{MissionID: "m-zzz-1", TaskTitles: []string{"Improve widget caching consistency layer"}},
		{MissionID: "m-zzz-2", TaskTitles: []string{"Improve widget caching consistency layer"}},
		{MissionID: "m-aaa-1", TaskTitles: []string{"Refactor authentication middleware pipeline"}},
		{MissionID: "m-aaa-2", TaskTitles: []string{"Refactor authentication middleware pipeline"}},
	}
	clusters := treasure.BuildClusters(missions)
	require.Len(t, clusters, 2)
	assert.Less(t, clusters[0].ID, clusters[1].ID, "clusters must be sorted by ID")
}

// --- treasure.ClusterID: truncates to first two tags ---

func TestClusterID_TruncatesToTwoTags(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "cluster-alpha-beta", treasure.ClusterID([]string{"alpha", "beta", "gamma"}))
}

// --- treasure.BuildGaps: same treasure.Gap id cited by multiple missions ---

func TestBuildGaps_DuplicateIDAcrossMissionsAppendsCitation(t *testing.T) {
	t.Parallel()
	missions := []treasure.ScannedMission{
		{MissionID: "mission-a", SQs: []treasure.SQEntry{{ID: "SQ-1", Status: "sq_pending"}}},
		{MissionID: "mission-b", SQs: []treasure.SQEntry{{ID: "SQ-1", Status: "sq_pending"}}},
	}
	gaps := treasure.BuildGaps(missions)
	require.Len(t, gaps, 1)
	assert.Equal(t, []string{"mission-a", "mission-b"}, gaps[0].CitedMissions)
}

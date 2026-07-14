package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Whitebox tests targeting uncovered branches in treasure_chest_scan.go: command-level
// error paths (resolveStrategistRoot/resolveDojoRoots/scanMissions/writeScanOutputs
// failures), permission-based I/O failures in the writer helpers, and pure-function edge
// cases (multi-cluster sort, duplicate gap citation, id truncation) not exercised by the
// happy-path tests in treasure_chest_scan_test.go.

// --- runTreasureChestScan: resolveStrategistRoot / resolveDojoRoots errors ---

func TestTreasureChestScan_ResolveRootError(t *testing.T) {
	resetTreasureChestFlags(t)
	treasureChestRoot = ""

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
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(": not: valid: yaml:\n"), 0o644))
	resetTreasureChestFlags(t)
	treasureChestRoot = dir

	err := treasureChestScanCmd.RunE(treasureChestScanCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest scan")
}

// --- runTreasureChestScan: scanMissions error (unreadable refined/ dir) ---

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
	treasureChestRoot = dir

	err := treasureChestScanCmd.RunE(treasureChestScanCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest scan")
}

// --- runTreasureChestScan: writeScanOutputs error (clustersDir parent unwritable) ---

func TestTreasureChestScan_WriteScanOutputsError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir, basePath := scanTestRoot(t)
	writeMissionTasks(t, basePath, "refined", "mission-a", "side_quests_approved:\n\n- id: `SQ-101`\n  description: Pending item.\n  status: `sq_pending`\n")
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	resetTreasureChestFlags(t)
	treasureChestRoot = dir

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
	treasureChestRoot = dir
	treasureChestScanDryRun = true
	t.Cleanup(func() { treasureChestScanDryRun = false })

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestScanCmd.RunE(treasureChestScanCmd, nil))
	})
	assert.Contains(t, out, "cluster: ")
	assert.Contains(t, out, "gap: ")
}

// --- scanMissionsInDir: readdir permission error ---

func TestScanMissionsInDir_ReadDirPermissionError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	sub := filepath.Join(dir, "refined")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	require.NoError(t, os.Chmod(sub, 0o000))
	t.Cleanup(func() { _ = os.Chmod(sub, 0o755) })

	_, err := scanMissionsInDir(sub)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read")
}

// --- scanMissionsInDir / parseMissionTasks: propagated parse error ---

func TestScanMissionsInDir_PropagatesParseError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	missionDir := filepath.Join(dir, "mission-a")
	require.NoError(t, os.MkdirAll(missionDir, 0o755))
	// side_quests_approved: block with invalid YAML underneath.
	require.NoError(t, os.WriteFile(filepath.Join(missionDir, "tasks.md"),
		[]byte("side_quests_approved:\n  : not: valid:\n"), 0o644))

	_, err := scanMissionsInDir(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse side_quests_approved")
}

func TestParseMissionTasks_InvalidYAML(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "tasks.md")
	require.NoError(t, os.WriteFile(path, []byte("side_quests_approved:\n  : not: valid:\n"), 0o644))

	_, err := parseMissionTasks("mission-x", path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse side_quests_approved")
}

func TestParseMissionTasks_MissingFile(t *testing.T) {
	t.Parallel()
	_, err := parseMissionTasks("mission-x", filepath.Join(t.TempDir(), "nope.md"))
	require.Error(t, err)
}

// --- regenerateDir ---

func TestRegenerateDir_MkdirAllError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	parent := t.TempDir()
	require.NoError(t, os.Chmod(parent, 0o555))
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })

	err := regenerateDir(filepath.Join(parent, "child"))
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

	err := regenerateDir(target)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remove")
}

func TestRegenerateDir_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, "child")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "stray"), []byte("x"), 0o644)) // unrelated file, unaffected
	require.NoError(t, regenerateDir(target))
	info, err := os.Stat(target)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// --- writeClusterFile / writeGapFile: write errors ---

func TestWriteClusterFile_WriteError(t *testing.T) {
	t.Parallel()
	// Directory does not exist -> os.WriteFile fails.
	err := writeClusterFile(filepath.Join(t.TempDir(), "nonexistent"), cluster{ID: "cluster-x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write")
}

func TestWriteGapFile_WriteError(t *testing.T) {
	t.Parallel()
	err := writeGapFile(filepath.Join(t.TempDir(), "nonexistent"), gap{ID: "sq-001"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write")
}

func TestWriteGapFile_WithDependencies(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, writeGapFile(dir, gap{ID: "sq-001", Dependencies: []string{"sq-000"}, Status: "sq_pending"}))
	raw, err := os.ReadFile(filepath.Join(dir, "sq-001.md"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "dependencies: [sq-000]")
}

// --- writeScanOutputs: gapsDir regeneration failure (second regenerateDir call) ---

func TestWriteScanOutputs_GapsDirError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	parent := t.TempDir()
	clustersDir := filepath.Join(parent, "clusters")
	gapsDir := filepath.Join(parent, "locked", "gaps")
	require.NoError(t, os.MkdirAll(filepath.Join(parent, "locked"), 0o555))
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(parent, "locked"), 0o755) })

	err := writeScanOutputs(clustersDir, nil, gapsDir, []gap{{ID: "sq-001", Status: "sq_pending"}})
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

	err := writeScanOutputs(clustersDir, []cluster{{ID: "cluster-x"}}, filepath.Join(parent, "gaps"), nil)
	require.Error(t, err)
}

// --- buildClusters: multiple clusters (exercises the sort.Slice comparator) ---

func TestBuildClusters_SortsMultipleClusters(t *testing.T) {
	t.Parallel()
	missions := []scannedMission{
		{MissionID: "m-zzz-1", TaskTitles: []string{"Improve widget caching consistency layer"}},
		{MissionID: "m-zzz-2", TaskTitles: []string{"Improve widget caching consistency layer"}},
		{MissionID: "m-aaa-1", TaskTitles: []string{"Refactor authentication middleware pipeline"}},
		{MissionID: "m-aaa-2", TaskTitles: []string{"Refactor authentication middleware pipeline"}},
	}
	clusters := buildClusters(missions)
	require.Len(t, clusters, 2)
	assert.Less(t, clusters[0].ID, clusters[1].ID, "clusters must be sorted by ID")
}

// --- clusterID: truncates to first two tags ---

func TestClusterID_TruncatesToTwoTags(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "cluster-alpha-beta", clusterID([]string{"alpha", "beta", "gamma"}))
}

// --- buildGaps: same gap id cited by multiple missions ---

func TestBuildGaps_DuplicateIDAcrossMissionsAppendsCitation(t *testing.T) {
	t.Parallel()
	missions := []scannedMission{
		{MissionID: "mission-a", SQs: []sqEntry{{ID: "SQ-1", Status: "sq_pending"}}},
		{MissionID: "mission-b", SQs: []sqEntry{{ID: "SQ-1", Status: "sq_pending"}}},
	}
	gaps := buildGaps(missions)
	require.Len(t, gaps, 1)
	assert.Equal(t, []string{"mission-a", "mission-b"}, gaps[0].CitedMissions)
}

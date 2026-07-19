package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scanTestRoot builds a project tree with a .strategist/ root and an .analysis/ base_path,
// matching resolveDojoRoots' expected layout (base_path resolved relative to the .strategist
// root's parent directory).
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

func writeMissionTasks(t *testing.T, basePath, category, missionID, content string) {
	t.Helper()
	dir := filepath.Join(basePath, category, missionID)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644))
}

func TestTreasureChestScan_ClustersMissionsWithSharedTags(t *testing.T) {
	dir, basePath := scanTestRoot(t)
	writeMissionTasks(t, basePath, "refined", "mission-a", "side_quests_approved:\n\n- id: `SQ-101`\n  description: Improve widget caching layer for faster loads.\n  status: `sq_pending`\n\n## Suggested Validation\n")
	writeMissionTasks(t, basePath, "refined", "mission-b", "## Task 1 — Improve widget rendering\n\nside_quests_approved:\n\n- id: `SQ-102`\n  description: Improve widget caching consistency.\n  status: `sq_closed_moot`\n\n## Suggested Validation\n")
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	require.NoError(t, treasureChestScanCmd.RunE(treasureChestScanCmd, nil))

	entries, err := os.ReadDir(filepath.Join(dir, "treasure", "clusters"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	content, err := os.ReadFile(filepath.Join(dir, "treasure", "clusters", entries[0].Name()))
	require.NoError(t, err)
	assert.Contains(t, string(content), "mission-a")
	assert.Contains(t, string(content), "mission-b")
}

func TestTreasureChestScan_NoClusterBelowTwoSharedTags(t *testing.T) {
	dir, basePath := scanTestRoot(t)
	writeMissionTasks(t, basePath, "refined", "mission-a", "side_quests_approved:\n\n- id: `SQ-101`\n  description: Widget caching improvements.\n  status: `sq_pending`\n")
	writeMissionTasks(t, basePath, "done", "mission-c", "side_quests_approved:\n\n- id: `SQ-103`\n  description: Totally unrelated documentation formatting topic.\n  status: `sq_pending`\n")
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	require.NoError(t, treasureChestScanCmd.RunE(treasureChestScanCmd, nil))

	entries, err := os.ReadDir(filepath.Join(dir, "treasure", "clusters"))
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestTreasureChestScan_GapsOnlyPendingStatus(t *testing.T) {
	dir, basePath := scanTestRoot(t)
	writeMissionTasks(t, basePath, "refined", "mission-a", "side_quests_approved:\n\n- id: `SQ-101`\n  description: Pending item.\n  status: `sq_pending`\n\n- id: `SQ-102`\n  description: Closed item.\n  status: `sq_closed_moot`\n")
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	require.NoError(t, treasureChestScanCmd.RunE(treasureChestScanCmd, nil))

	entries, err := os.ReadDir(filepath.Join(dir, "treasure", "gaps"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "sq-101.md", entries[0].Name())
}

func TestTreasureChestScan_ExcludesPendingScrapsAndArchivedReports(t *testing.T) {
	dir, basePath := scanTestRoot(t)
	// Not under refined/ or done/ — must never be read.
	require.NoError(t, os.MkdirAll(filepath.Join(basePath, "pending", "scraps"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(basePath, "pending", "scraps", "idea.md"), []byte("side_quests_approved:\n\n- id: `SQ-999`\n  status: `sq_pending`\n"), 0o644))
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	require.NoError(t, treasureChestScanCmd.RunE(treasureChestScanCmd, nil))

	entries, err := os.ReadDir(filepath.Join(dir, "treasure", "gaps"))
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestTreasureChestScan_DryRunWritesNothing(t *testing.T) {
	dir, basePath := scanTestRoot(t)
	writeMissionTasks(t, basePath, "refined", "mission-a", "side_quests_approved:\n\n- id: `SQ-101`\n  description: Pending item.\n  status: `sq_pending`\n")
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)
	setCmdFlag(t, treasureChestScanCmd, "dry-run", "true")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestScanCmd.RunE(treasureChestScanCmd, nil))
	})
	assert.Contains(t, out, "dry-run")
	_, err := os.Stat(filepath.Join(dir, "treasure", "gaps"))
	assert.True(t, os.IsNotExist(err))
}

func TestTreasureChestScan_RegeneratesFromScratch(t *testing.T) {
	dir, basePath := scanTestRoot(t)
	writeMissionTasks(t, basePath, "refined", "mission-a", "side_quests_approved:\n\n- id: `SQ-101`\n  description: Pending item.\n  status: `sq_pending`\n")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "treasure", "gaps"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure", "gaps", "stale-gap.md"), []byte("stale"), 0o644))
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	require.NoError(t, treasureChestScanCmd.RunE(treasureChestScanCmd, nil))

	_, err := os.Stat(filepath.Join(dir, "treasure", "gaps", "stale-gap.md"))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(dir, "treasure", "gaps", "sq-101.md"))
	assert.NoError(t, err)
}

func TestExtractTags_FiltersShortWordsAndStopwords(t *testing.T) {
	t.Parallel()
	m := treasure.ScannedMission{TaskTitles: []string{"Improve the widget caching with this"}}
	tags := treasure.ExtractTags(m)
	assert.Contains(t, tags, "widget")
	assert.Contains(t, tags, "caching")
	assert.Contains(t, tags, "improve")
	assert.NotContains(t, tags, "the")
	assert.NotContains(t, tags, "with")
	assert.NotContains(t, tags, "this")
}

func TestGapID_Lowercases(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "sq-005", treasure.GapID("SQ-005"))
}

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// indexTestRoot builds a full .strategist/-like tree (personas/roles/knowledge index for
// CompileAll, plus treasure-chests.yaml and jewels.yaml) with a sibling .analysis/ base_path,
// matching resolveDojoRoots' expected layout.
func indexTestRoot(t *testing.T) (strategistDir, basePath string) {
	t.Helper()
	root := t.TempDir()
	strategistDir = filepath.Join(root, ".strategist")
	basePath = filepath.Join(root, ".analysis")
	require.NoError(t, os.MkdirAll(strategistDir, 0o755))
	testutil.MinimalRoot(t, strategistDir)

	require.NoError(t, os.WriteFile(filepath.Join(strategistDir, "treasure-chests.yaml"), []byte(`
schema_version: "1"
chests: []
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(strategistDir, "jewels.yaml"), []byte(`
schema_version: "1"
jewels: []
`), 0o644))
	return strategistDir, basePath
}

func TestTreasureChestIndex_GeneratesProposedJewelsFromClusters(t *testing.T) {
	dir, basePath := indexTestRoot(t)
	writeMissionTasks(t, basePath, "refined", "mission-a", "side_quests_approved:\n\n- id: `SQ-101`\n  description: Improve widget caching layer for faster loads.\n  status: `sq_pending`\n\n## Suggested Validation\n")
	writeMissionTasks(t, basePath, "refined", "mission-b", "## Task 1 — Improve widget rendering\n\nside_quests_approved:\n\n- id: `SQ-102`\n  description: Improve widget caching consistency.\n  status: `sq_closed_moot`\n\n## Suggested Validation\n")
	resetTreasureChestFlags(t)
	treasureChestRoot = dir

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestIndexCmd.RunE(treasureChestIndexCmd, nil))
	})
	assert.Contains(t, out, "proposed jewel(s) written")
	assert.FileExists(t, filepath.Join(dir, ".compiled", ".index.gz"))

	raw, err := os.ReadFile(filepath.Join(dir, "jewels.yaml"))
	require.NoError(t, err)
	content := string(raw)
	assert.Contains(t, content, "chest_id: mission-history")
	assert.Contains(t, content, "status: proposed")
	assert.Contains(t, content, "kind: pattern")
}

func TestTreasureChestIndex_GeneratesProposedJewelFromGap(t *testing.T) {
	dir, basePath := indexTestRoot(t)
	writeMissionTasks(t, basePath, "refined", "mission-a", "side_quests_approved:\n\n- id: `SQ-101`\n  description: Pending item.\n  status: `sq_pending`\n")
	resetTreasureChestFlags(t)
	treasureChestRoot = dir

	require.NoError(t, treasureChestIndexCmd.RunE(treasureChestIndexCmd, nil))

	raw, err := os.ReadFile(filepath.Join(dir, "jewels.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "id: jewel-gap-sq-101")
	assert.Contains(t, string(raw), "kind: gap")
}

func TestTreasureChestIndex_DeduplicatesAgainstExistingJewels(t *testing.T) {
	dir, basePath := indexTestRoot(t)
	writeMissionTasks(t, basePath, "refined", "mission-a", "side_quests_approved:\n\n- id: `SQ-101`\n  description: Pending item.\n  status: `sq_pending`\n")
	resetTreasureChestFlags(t)
	treasureChestRoot = dir

	// First run creates the proposed gap jewel.
	require.NoError(t, treasureChestIndexCmd.RunE(treasureChestIndexCmd, nil))
	raw, err := os.ReadFile(filepath.Join(dir, "jewels.yaml"))
	require.NoError(t, err)
	firstCount := countOccurrences(string(raw), "id: jewel-gap-sq-101")
	require.Equal(t, 1, firstCount)

	// Second run must not duplicate the same candidate id.
	out := captureStdout(t, func() {
		require.NoError(t, treasureChestIndexCmd.RunE(treasureChestIndexCmd, nil))
	})
	assert.Contains(t, out, "1 duplicate(s) skipped")

	raw, err = os.ReadFile(filepath.Join(dir, "jewels.yaml"))
	require.NoError(t, err)
	assert.Equal(t, 1, countOccurrences(string(raw), "id: jewel-gap-sq-101"))
}

func TestTreasureChestIndex_NoCandidatesLeavesJewelsUntouched(t *testing.T) {
	dir, _ := indexTestRoot(t)
	resetTreasureChestFlags(t)
	treasureChestRoot = dir

	before, err := os.ReadFile(filepath.Join(dir, "jewels.yaml"))
	require.NoError(t, err)

	require.NoError(t, treasureChestIndexCmd.RunE(treasureChestIndexCmd, nil))

	after, err := os.ReadFile(filepath.Join(dir, "jewels.yaml"))
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after))
}

func countOccurrences(s, substr string) int {
	n := 0
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			n++
			i += len(substr) - 1
		}
	}
	return n
}

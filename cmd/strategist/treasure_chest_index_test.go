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
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestIndexCmd.RunE(treasureChestIndexCmd, nil))
	})
	assert.Contains(t, out, "proposed jewel(s) written")
	assert.FileExists(t, filepath.Join(dir, ".compiled", ".index.gz"))

	raw, err := os.ReadFile(filepath.Join(dir, "jewels", "mission-history.yaml"))
	require.NoError(t, err)
	content := string(raw)
	assert.Contains(t, content, "chest_id: mission-history")
	assert.Contains(t, content, "status: proposed")
	assert.Contains(t, content, "kind: pattern")
	assert.Contains(t, content, "history:")
	assert.Contains(t, content, "by: agent")
}

func TestTreasureChestIndex_GeneratesProposedJewelFromGap(t *testing.T) {
	dir, basePath := indexTestRoot(t)
	writeMissionTasks(t, basePath, "refined", "mission-a", "side_quests_approved:\n\n- id: `SQ-101`\n  description: Pending item.\n  status: `sq_pending`\n")
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	require.NoError(t, treasureChestIndexCmd.RunE(treasureChestIndexCmd, nil))

	raw, err := os.ReadFile(filepath.Join(dir, "jewels", "mission-history.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "id: jewel-gap-sq-101")
	assert.Contains(t, string(raw), "kind: gap")
}

func TestTreasureChestIndex_AcceptsFencedSideQuestsYAML(t *testing.T) {
	dir, basePath := indexTestRoot(t)
	writeMissionTasks(t, basePath, "done", "mission-a", `
## Archivist-To-Sniper Handoff Fields

side_quests_approved:

`+"```yaml"+`
- id: SQ-101
  description: Pending fenced side quest.
  strategy: execute_later
  estimated_impact: low
  dependencies: []
  status: sq_pending
`+"```"+`

acceptance_checks:

`+"```yaml"+`
- "fenced acceptance checks must not be parsed as side quests"
`+"```"+`
`)
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	require.NoError(t, treasureChestIndexCmd.RunE(treasureChestIndexCmd, nil))

	raw, err := os.ReadFile(filepath.Join(dir, "jewels", "mission-history.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "id: jewel-gap-sq-101")
}

func TestTreasureChestIndex_WarnsAndContinuesOnInvalidMissionTasks(t *testing.T) {
	dir, basePath := indexTestRoot(t)
	writeMissionTasks(t, basePath, "refined", "invalid-mission", "side_quests_approved:\n  : not: valID:\n")
	writeMissionTasks(t, basePath, "refined", "valid-mission", "side_quests_approved:\n\n- id: SQ-201\n  description: Pending valid item.\n  status: sq_pending\n")
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	errOut := captureStderr(t, func() {
		require.NoError(t, treasureChestIndexCmd.RunE(treasureChestIndexCmd, nil))
	})
	assert.Contains(t, errOut, "Warning: treasure-chest index: skipped inconsistent mission file")
	assert.Contains(t, errOut, filepath.Join("invalid-mission", "tasks.md"))
	assert.Contains(t, errOut, "parse side_quests_approved")

	raw, err := os.ReadFile(filepath.Join(dir, "jewels", "mission-history.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "id: jewel-gap-sq-201")
}

func TestTreasureChestIndex_UsesConfiguredScoringPolicy(t *testing.T) {
	dir, basePath := indexTestRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte(`
schema_version: "1"
scoring_policy:
  gap_base: 7
  gap_mission_weight: 2
chests: []
`), 0o644))
	writeMissionTasks(t, basePath, "refined", "mission-a", "side_quests_approved:\n\n- id: `SQ-101`\n  description: Pending item.\n  status: `sq_pending`\n")
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	require.NoError(t, treasureChestIndexCmd.RunE(treasureChestIndexCmd, nil))

	raw, err := os.ReadFile(filepath.Join(dir, "jewels", "mission-history.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "value: 9")
}

func TestTreasureChestIndex_DeduplicatesAgainstExistingJewels(t *testing.T) {
	dir, basePath := indexTestRoot(t)
	writeMissionTasks(t, basePath, "refined", "mission-a", "side_quests_approved:\n\n- id: `SQ-101`\n  description: Pending item.\n  status: `sq_pending`\n")
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	// First run creates the proposed treasure.Gap jewel.
	require.NoError(t, treasureChestIndexCmd.RunE(treasureChestIndexCmd, nil))
	raw, err := os.ReadFile(filepath.Join(dir, "jewels", "mission-history.yaml"))
	require.NoError(t, err)
	firstCount := countOccurrences(string(raw), "id: jewel-gap-sq-101")
	require.Equal(t, 1, firstCount)

	// Second run must not duplicate the same candidate id.
	out := captureStdout(t, func() {
		require.NoError(t, treasureChestIndexCmd.RunE(treasureChestIndexCmd, nil))
	})
	assert.Contains(t, out, "1 duplicate(s) skipped")

	raw, err = os.ReadFile(filepath.Join(dir, "jewels", "mission-history.yaml"))
	require.NoError(t, err)
	assert.Equal(t, 1, countOccurrences(string(raw), "id: jewel-gap-sq-101"))
}

func TestTreasureChestIndex_NoCandidatesLeavesJewelsUntouched(t *testing.T) {
	dir, _ := indexTestRoot(t)
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	before, err := os.ReadFile(filepath.Join(dir, "jewels.yaml"))
	require.NoError(t, err)

	require.NoError(t, treasureChestIndexCmd.RunE(treasureChestIndexCmd, nil))

	after, err := os.ReadFile(filepath.Join(dir, "jewels.yaml"))
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after))
}

func TestTreasureChestIndex_GeneratesProposedPotionsFromRunbooksChest(t *testing.T) {
	dir, _ := indexTestRoot(t)
	runbooksDir := filepath.Join(filepath.Dir(dir), "docs", "runbooks")
	require.NoError(t, os.MkdirAll(runbooksDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(runbooksDir, "sample-issue.md"), []byte(
		"# Runbook: Sample Issue\n\n## Symptom\n\nSomething breaks when X happens.\n\n## Resolution Steps\n\n1. Do Y.\n"),
		0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte(`
schema_version: "1"
chests:
  - id: runbooks
    title: Agent Runbooks
    path: docs/runbooks
    trust:
      tier: T2
    routing:
      task_types: [all]
`), 0o644))
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestIndexCmd.RunE(treasureChestIndexCmd, nil))
	})
	assert.Contains(t, out, "1 proposed potion(s) written")

	raw, err := os.ReadFile(filepath.Join(dir, "potions", "runbooks.yaml"))
	require.NoError(t, err)
	content := string(raw)
	assert.Contains(t, content, "id: potion-sample-issue")
	assert.Contains(t, content, "status: proposed")
	assert.Contains(t, content, "Something breaks when X happens.")
}

func TestTreasureChestIndex_NoRunbooksChestWritesNoPotions(t *testing.T) {
	dir, _ := indexTestRoot(t)
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	require.NoError(t, treasureChestIndexCmd.RunE(treasureChestIndexCmd, nil))

	_, err := os.Stat(filepath.Join(dir, "potions", "runbooks.yaml"))
	assert.True(t, os.IsNotExist(err))
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

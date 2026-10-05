package treasurecli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const parityJewelYAML = `
schema_version: "1"
jewels:
  - id: jewel-1
    chest_id: source
    kind: pattern
    statement: "Widgets require explicit teardown."
    source_refs: ["source#widgets"]
    trust: T1
    status: proposed
    reviewed_by: agent
    score: {value: 55, reasons: ["recurring across 2 missions"]}
`

const parityPotionYAML = `
schema_version: "1"
potions:
  - id: potion-1
    chest_id: runbooks
    runbook_ref: docs/runbooks/sample.md
    when_to_use: "When sample breaks."
    trust: T2
    status: proposed
    source_refs: ["docs/runbooks/sample.md"]
    reviewed_by: agent
`

func parityFixture(t *testing.T) string {
	t.Helper()
	root := minimalTreasureChestRoot(t)
	project := filepath.Dir(root)
	require.NoError(t, os.MkdirAll(filepath.Join(project, ".analysis", "refined"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(project, ".analysis", "done"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(project, "docs", "runbooks"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(project, "docs", "runbooks", "sample.md"), []byte(
		"# Runbook: Sample\n\n## Symptom\n\nSomething breaks.\n\n## Resolution Steps\n\n1. Do Y.\n",
	), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte(`# preserved active comment
mode: epic
base_path: .analysis
slots:
  discovery: brainstorming
  refinement: openspec-explore
  execution: sniper
treasure_chests:
  - id: source
    path: .sdd/source
    scope: all
  - id: runbooks
    path: docs/runbooks
    scope: all
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "treasure-chests.yaml"), []byte(`# preserved governance comment
schema_version: "1"
chests:
  - id: source
    title: Source
    path: .sdd/source
    trust: {tier: T1, reviewed_by: human, last_reviewed: 2026-06-24}
  - id: runbooks
    title: Runbooks
    path: docs/runbooks
    trust: {tier: T2, reviewed_by: human, last_reviewed: 2026-06-24}
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "knowledge.index.yaml"), []byte(`
sources:
  - id: source
    path: .sdd/source
    tags: [all]
  - id: runbooks
    path: docs/runbooks
    tags: [all]
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "jewels.yaml"), []byte(parityJewelYAML), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "potions.yaml"), []byte(parityPotionYAML), 0o644))
	return root
}

func parityMigrationFixture(t *testing.T) string {
	t.Helper()
	root := parityFixture(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "jewels.yaml"), []byte(`
schema_version: "1"
jewels:
  - id: legacy-jewel
    chest_id: source
    kind: pattern
    statement: "Legacy status is migrated."
    source_refs: ["source#legacy"]
    trust: T1
    status: active
    reviewed_by: agent
`), 0o644))
	return root
}

func parityScanFixture(t *testing.T) string {
	t.Helper()
	root := parityFixture(t)
	missionDir := filepath.Join(filepath.Dir(root), ".analysis", "refined", "mission-a")
	require.NoError(t, os.MkdirAll(missionDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(missionDir, "tasks.md"), []byte(
		"side_quests_approved:\n\n- id: `SQ-101`\n  description: Widget caching improvement.\n  status: `sq_pending`\n",
	), 0o644))
	return root
}

func paritySnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	snapshot := make(map[string]string)
	require.NoError(t, filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		require.NoError(t, err)
		if info.IsDir() {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		rel, relErr := filepath.Rel(root, path)
		require.NoError(t, relErr)
		snapshot[rel] = string(raw)
		return nil
	}))
	return snapshot
}

package treasure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func minimalChestRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: epic
base_path: .analysis
slots:
  discovery: brainstorming
  refinement: openspec-explore
  execution: sdd-ask
treasure_chests:
  - id: source
    path: .sdd/source
    scope: all
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte(`
schema_version: "1"
chests:
  - id: source
    title: Source
    path: .sdd/source
    trust:
      tier: T1
      reviewed_by: human
      last_reviewed: 2026-06-24
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "knowledge.index.yaml"), []byte(`
sources:
  - id: source
    path: .sdd/source
    tags: [all]
`), 0o644))

	return dir
}

func TestExecuteAdd_WritesAllThreeFiles(t *testing.T) {
	dir := minimalChestRoot(t)

	indexPath, err := ExecuteAdd(dir, AddOptions{
		ID:         "new-chest",
		Path:       "/tmp/new-chest",
		Scope:      "all",
		TrustTier:  "T1",
		ReviewedBy: "human",
		Tags:       []string{"all"},
	})
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "knowledge.index.yaml"), indexPath)

	active, err := LoadActiveChests(dir)
	require.NoError(t, err)
	require.Len(t, active, 2)
	assert.Equal(t, "new-chest", active[1].ID)
	assert.Equal(t, "/tmp/new-chest", active[1].Path)

	governed, err := LoadGoverned(dir)
	require.NoError(t, err)
	require.Contains(t, governed, "new-chest")
	assert.Equal(t, "T1", governed["new-chest"].Trust.Tier)

	indexed, err := LoadIndexed(dir)
	require.NoError(t, err)
	assert.True(t, indexed["new-chest"])
}

func TestExecuteAdd_MissingGovernedFileErrors(t *testing.T) {
	dir := minimalChestRoot(t)
	require.NoError(t, os.Remove(filepath.Join(dir, "treasure-chests.yaml")))

	_, err := ExecuteAdd(dir, AddOptions{ID: "new-chest", Path: "/tmp/new-chest", Scope: "all", TrustTier: "T1", ReviewedBy: "human"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load chest YAML docs")
}

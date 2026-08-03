package treasure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadScoringPolicy_DefaultWhenMissing(t *testing.T) {
	t.Parallel()

	got, err := LoadScoringPolicy(t.TempDir())

	require.NoError(t, err)
	assert.Equal(t, DefaultScoringPolicy(), got)
}

func TestLoadScoringPolicy_PartialOverrides(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte(`
schema_version: "1"
scoring_policy:
  cluster_base: 12
  gap_mission_weight: 3
chests: []
`), 0o644))

	got, err := LoadScoringPolicy(dir)

	require.NoError(t, err)
	expected := DefaultScoringPolicy()
	expected.ClusterBase = 12
	expected.GapMissionWeight = 3
	assert.Equal(t, expected, got)
}

func TestLoadScoringPolicy_InvalidPolicyErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte(`
schema_version: "1"
scoring_policy:
  cluster_tag_weight: -1
chests: []
`), 0o644))

	_, err := LoadScoringPolicy(dir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cluster_tag_weight")
}

func TestLoadIndexed_NotExist(t *testing.T) {
	t.Parallel()
	result, err := LoadIndexed(t.TempDir())
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestLoadIndexed_CorruptYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "knowledge.index.yaml"),
		[]byte(": not: valID:\n"), 0o644))

	_, err := LoadIndexed(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "knowledge.index.yaml")
}

func TestLoadIndexed_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "knowledge.index.yaml"),
		[]byte("sources:\n  - id: chest-x\n  - id: chest-y\n"), 0o644))

	result, err := LoadIndexed(dir)
	require.NoError(t, err)
	assert.True(t, result["chest-x"])
	assert.True(t, result["chest-y"])
}

func TestLoadIndexed_TreatsSourceIDsAsDataNotPaths(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "knowledge.index.yaml"),
		[]byte("sources:\n  - id: ../outside\n  - id: /tmp/absolute\n"), 0o644))

	result, err := LoadIndexed(dir)

	require.NoError(t, err)
	assert.True(t, result["../outside"])
	assert.True(t, result["/tmp/absolute"])
}

func TestLoadActiveChests_CorruptYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"),
		[]byte(": not: valID: yaml:\n"), 0o644))

	_, err := LoadActiveChests(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "active.yaml")
}

func TestLoadActiveChests_TreatsChestPathAsDataNotReadTarget(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
treasure_chests:
  - id: suspicious
    path: ../outside.md
    scope: discovery
`), 0o644))

	got, err := LoadActiveChests(dir)

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "../outside.md", got[0].Path)
	assert.Equal(t, Scope{"discovery"}, got[0].Scope)
}

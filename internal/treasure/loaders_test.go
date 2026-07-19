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

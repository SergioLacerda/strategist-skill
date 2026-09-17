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

func TestLoadScoringPolicy_ParseErrorPropagates(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte(": not: valid: yaml:\n"), 0o644))

	_, err := LoadScoringPolicy(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse treasure-chests.yaml")
}

func TestLoadScoringPolicy_ReadErrorPropagates(t *testing.T) {
	skipIfPermissionTestUnsupported(t)
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o000))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	_, err := LoadScoringPolicy(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read treasure-chests.yaml")
}

// TestLoadScoringPolicy_ReadErrorPropagates_NotIsNotExist covers the same
// os.ReadFile error branch as TestLoadScoringPolicy_ReadErrorPropagates, but
// portably (no chmod, so it isn't skipped on platforms where permission bits
// don't produce a read error): treasure-chests.yaml is a directory, not a
// file, so os.ReadFile fails with an error that is not os.IsNotExist.
func TestLoadScoringPolicy_ReadErrorPropagates_NotIsNotExist(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "treasure-chests.yaml"), 0o755))

	_, err := LoadScoringPolicy(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read treasure-chests.yaml")
}

// TestLoadScoringPolicy_RemainingOverrideFields covers the three
// scoring_policy override fields TestLoadScoringPolicy_PartialOverrides
// doesn't exercise (cluster_mission_weight, gap_base, max_score) — the
// other three (cluster_base, cluster_tag_weight, gap_mission_weight) are
// already covered by TestLoadScoringPolicy_PartialOverrides and
// TestLoadScoringPolicy_InvalidPolicyErrors.
func TestLoadScoringPolicy_RemainingOverrideFields(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte(`
schema_version: "1"
scoring_policy:
  cluster_mission_weight: 7
  gap_base: 20
  max_score: 50
chests: []
`), 0o644))

	got, err := LoadScoringPolicy(dir)

	require.NoError(t, err)
	expected := DefaultScoringPolicy()
	expected.ClusterMissionWeight = 7
	expected.GapBase = 20
	expected.MaxScore = 50
	assert.Equal(t, expected, got)
}

func TestValidateScoringPolicy_EachNegativeWeightErrors(t *testing.T) {
	t.Parallel()
	base := DefaultScoringPolicy()

	cases := []struct {
		name   string
		mutate func(*ScoringPolicy)
	}{
		{"cluster_base", func(p *ScoringPolicy) { p.ClusterBase = -1 }},
		{"cluster_mission_weight", func(p *ScoringPolicy) { p.ClusterMissionWeight = -1 }},
		{"gap_base", func(p *ScoringPolicy) { p.GapBase = -1 }},
		{"gap_mission_weight", func(p *ScoringPolicy) { p.GapMissionWeight = -1 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := base
			tc.mutate(&p)
			err := ValidateScoringPolicy(p)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.name)
		})
	}
}

func TestValidateScoringPolicy_MaxScoreOutOfRangeErrors(t *testing.T) {
	t.Parallel()
	p := DefaultScoringPolicy()
	p.MaxScore = 0
	require.Error(t, ValidateScoringPolicy(p))

	p.MaxScore = 101
	require.Error(t, ValidateScoringPolicy(p))
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

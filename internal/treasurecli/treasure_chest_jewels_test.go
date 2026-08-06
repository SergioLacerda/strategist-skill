package treasurecli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadJewels_NotExist(t *testing.T) {
	t.Parallel()
	result, err := treasure.LoadJewels(t.TempDir(), nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestLoadJewels_ValidEntry(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "jewels.yaml"), []byte(`
schema_version: "1"
jewels:
  - id: jewel-1
    chest_id: source
    kind: pattern
    statement: "Widgets require explicit teardown."
    source_refs: ["source#widgets"]
    trust: T1
    status: accepted
    reviewed_by: agent
`), 0o644))
	governed := map[string]treasure.GovernedChest{
		"source": {ID: "source", Trust: treasure.GovernedTrust{Tier: "T1"}},
	}

	result, err := treasure.LoadJewels(dir, governed)
	require.NoError(t, err)
	require.Contains(t, result, "source")
	require.Len(t, result["source"], 1)
	assert.Equal(t, "jewel-1", result["source"][0].ID)
}

func TestLoadJewels_PartitionedEntries(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "jewels"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "jewels", "source.yaml"), []byte(`
schema_version: "1"
jewels:
  - id: jewel-1
    chest_id: source
    kind: pattern
    statement: "Partitioned jewel."
    source_refs: ["source#widgets"]
    trust: T1
    status: accepted
    reviewed_by: agent
`), 0o644))

	result, err := treasure.LoadJewels(dir, nil)
	require.NoError(t, err)
	require.Contains(t, result, "source")
	require.Len(t, result["source"], 1)
	assert.Equal(t, "jewel-1", result["source"][0].ID)
}

func TestLoadJewels_MixedLayoutDeduplicatesByID(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "jewels.yaml"), []byte(`
schema_version: "1"
jewels:
  - id: jewel-1
    chest_id: source
    kind: pattern
    statement: "Monolithic jewel wins."
    source_refs: ["source#mono"]
    trust: T1
    status: accepted
    reviewed_by: agent
`), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "jewels"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "jewels", "source.yaml"), []byte(`
schema_version: "1"
jewels:
  - id: jewel-1
    chest_id: source
    kind: pattern
    statement: "Duplicate partitioned jewel."
    source_refs: ["source#partition"]
    trust: T1
    status: accepted
    reviewed_by: agent
`), 0o644))

	result, err := treasure.LoadJewels(dir, nil)
	require.NoError(t, err)
	require.Len(t, result["source"], 1)
	assert.Equal(t, []string{"source#mono"}, result["source"][0].SourceRefs)
}

func TestLoadJewels_TrustExceedsParentChestTierErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "jewels.yaml"), []byte(`
schema_version: "1"
jewels:
  - id: jewel-1
    chest_id: source
    kind: pattern
    statement: "Over-trusted jewel."
    source_refs: ["source#x"]
    trust: T0
    status: accepted
    reviewed_by: agent
`), 0o644))
	governed := map[string]treasure.GovernedChest{
		"source": {ID: "source", Trust: treasure.GovernedTrust{Tier: "T2"}},
	}

	_, err := treasure.LoadJewels(dir, governed)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds parent chest's trust tier")
}

func TestLoadJewels_MissingChestIDErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "jewels.yaml"), []byte(`
schema_version: "1"
jewels:
  - id: jewel-1
    statement: "No parent."
    source_refs: ["source#x"]
    trust: T1
`), 0o644))

	_, err := treasure.LoadJewels(dir, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing chest_id")
}

func TestLoadJewels_MissingSourceRefsErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "jewels.yaml"), []byte(`
schema_version: "1"
jewels:
  - id: jewel-1
    chest_id: source
    statement: "No sources."
    trust: T1
`), 0o644))

	_, err := treasure.LoadJewels(dir, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing source_refs")
}

func TestNonDeprecatedJewelCount_ExcludesDeprecated(t *testing.T) {
	t.Parallel()
	jewels := []treasure.Jewel{
		{ID: "a", Status: "proposed"},
		{ID: "b", Status: "deprecated"},
		{ID: "c", Status: "accepted"},
	}
	assert.Equal(t, 2, treasure.NonDeprecatedJewelCount(jewels))
}

func TestLoadJewels_LegacyActiveStatusErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "jewels.yaml"), []byte(`
schema_version: "1"
jewels:
  - id: jewel-1
    chest_id: source
    kind: pattern
    statement: "Pre-migration jewel."
    source_refs: ["source#x"]
    trust: T1
    status: active
    reviewed_by: agent
`), 0o644))

	_, err := treasure.LoadJewels(dir, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "migrate-status")
}

func TestLoadJewels_UnknownStatusErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "jewels.yaml"), []byte(`
schema_version: "1"
jewels:
  - id: jewel-1
    chest_id: source
    kind: pattern
    statement: "Bad status."
    source_refs: ["source#x"]
    trust: T1
    status: bogus
    reviewed_by: agent
`), 0o644))

	_, err := treasure.LoadJewels(dir, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be one of proposed, accepted, verified, deprecated")
}

func TestTreasureChestCmd_ShowsJewelsColumn(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "jewels.yaml"), []byte(`
schema_version: "1"
jewels:
  - id: jewel-1
    chest_id: source
    kind: pattern
    statement: "A useful fact."
    source_refs: ["source#x"]
    trust: T1
    status: accepted
    reviewed_by: agent
`), 0o644))
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	})
	assert.Contains(t, out, "JEWELS")
	assert.Contains(t, out, "1")
}

func TestTreasureChestCmd_FormatJSON_IncludesJewelCount(t *testing.T) {
	dir := minimalTreasureChestRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "jewels.yaml"), []byte(`
schema_version: "1"
jewels:
  - id: jewel-1
    chest_id: source
    kind: pattern
    statement: "A useful fact."
    source_refs: ["source#x"]
    trust: T1
    status: accepted
    reviewed_by: agent
`), 0o644))
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)
	setTreasureChestFormat(t, "json")

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	})
	assert.Contains(t, out, `"jewel_count": 1`)
}

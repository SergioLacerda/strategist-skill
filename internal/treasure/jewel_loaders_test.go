package treasure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadJewels_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeJewelsFileT(t, dir, oneJewelYAML)

	got, err := LoadJewels(dir, nil)
	require.NoError(t, err)
	require.Len(t, got["source"], 1)
	assert.Equal(t, "jewel-1", got["source"][0].ID)
}

func TestLoadJewels_MissingFileIsNotError(t *testing.T) {
	t.Parallel()
	got, err := LoadJewels(t.TempDir(), nil)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestLoadJewels_MissingSourceRefsErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeJewelsFileT(t, dir, `
schema_version: "1"
jewels:
  - id: jewel-1
    chest_id: source
    kind: pattern
    statement: "x"
    trust: T1
    status: proposed
    reviewed_by: agent
`)
	_, err := LoadJewels(dir, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing source_refs")
}

func TestLoadJewels_TrustExceedsChestTierErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeJewelsFileT(t, dir, oneJewelYAML)
	governed := map[string]GovernedChest{
		"source": {ID: "source", Trust: GovernedTrust{Tier: "T2"}},
	}
	_, err := LoadJewels(dir, governed)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds parent chest's trust tier")
}

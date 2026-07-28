package treasure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const onePotionYAML = `
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

func writePotionsFileT(t *testing.T, dir, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "potions.yaml"), []byte(content), 0o644))
}

// --- loading / validation ---

func TestLoadPotions_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writePotionsFileT(t, dir, onePotionYAML)

	got, err := LoadPotions(dir, nil)
	require.NoError(t, err)
	require.Len(t, got["runbooks"], 1)
	assert.Equal(t, "potion-1", got["runbooks"][0].ID)
}

func TestLoadPotions_MissingFileIsNotError(t *testing.T) {
	t.Parallel()
	got, err := LoadPotions(t.TempDir(), nil)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestLoadPotions_MissingRunbookRefErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writePotionsFileT(t, dir, `
schema_version: "1"
potions:
  - id: potion-1
    chest_id: runbooks
    when_to_use: "When sample breaks."
    trust: T2
    status: proposed
    source_refs: ["docs/runbooks/sample.md"]
    reviewed_by: agent
`)

	_, err := LoadPotions(dir, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing runbook_ref")
}

func TestLoadPotions_TrustExceedsChestTierErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writePotionsFileT(t, dir, onePotionYAML)
	governed := map[string]GovernedChest{
		"runbooks": {ID: "runbooks", Trust: GovernedTrust{Tier: "T3"}},
	}

	_, err := LoadPotions(dir, governed)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds parent chest's trust tier")
}

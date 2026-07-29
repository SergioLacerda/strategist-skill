package treasure

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- lifecycle ---

func TestPromotePotion_Accept(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writePotionsFileT(t, dir, onePotionYAML)

	require.NoError(t, PromotePotion(dir, "potion-1", domain.PotionStatusAccepted, "", time.Now()))

	raw, err := os.ReadFile(filepath.Join(dir, "potions.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "status: accepted")
	assert.Contains(t, string(raw), "reviewed_by: human")
}

func TestPromotePotion_CannotPromoteDeprecated(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writePotionsFileT(t, dir, `
schema_version: "1"
potions:
  - id: potion-1
    chest_id: runbooks
    runbook_ref: docs/runbooks/sample.md
    when_to_use: "When sample breaks."
    trust: T2
    status: deprecated
    source_refs: ["docs/runbooks/sample.md"]
    reviewed_by: human
`)

	err := PromotePotion(dir, "potion-1", domain.PotionStatusAccepted, "", time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deprecated")
}

func TestPromotePotion_NotFound(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writePotionsFileT(t, dir, onePotionYAML)

	err := PromotePotion(dir, "does-not-exist", domain.PotionStatusAccepted, "", time.Now())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPotionNotFound)
}

package treasure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- index write ---

func TestWriteProposedPotions_WritesNewCandidates(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	candidates := []Potion{
		{ID: "potion-1", ChestID: "runbooks", RunbookRef: "docs/runbooks/a.md", WhenToUse: "a", Trust: "T2", Status: domain.PotionStatusProposed, SourceRefs: []string{"docs/runbooks/a.md"}, ReviewedBy: "agent"},
	}

	written, skipped, err := WriteProposedPotions(dir, candidates)
	require.NoError(t, err)
	assert.Equal(t, 1, written)
	assert.Equal(t, 0, skipped)

	raw, err := os.ReadFile(filepath.Join(dir, "potions", "runbooks.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "id: potion-1")
	assert.Contains(t, string(raw), "status: proposed")
}

func TestWriteProposedPotions_SkipsExistingIDs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writePotionsFileT(t, dir, onePotionYAML)
	candidates := []Potion{
		{ID: "potion-1", ChestID: "runbooks", RunbookRef: "docs/runbooks/sample.md", WhenToUse: "dup", Trust: "T2", Status: domain.PotionStatusProposed, SourceRefs: []string{"docs/runbooks/sample.md"}, ReviewedBy: "agent"},
	}

	written, skipped, err := WriteProposedPotions(dir, candidates)
	require.NoError(t, err)
	assert.Equal(t, 0, written)
	assert.Equal(t, 1, skipped)
}

func TestWriteProposedPotions_EmptyCandidatesNoop(t *testing.T) {
	t.Parallel()
	written, skipped, err := WriteProposedPotions(t.TempDir(), nil)
	require.NoError(t, err)
	assert.Equal(t, 0, written)
	assert.Equal(t, 0, skipped)
}

func TestExistingPotionIDs_NoManifestsIsEmpty(t *testing.T) {
	t.Parallel()
	got, err := ExistingPotionIDs(t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestExistingPotionIDs_CollectsAcrossManifest(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writePotionsFileT(t, dir, onePotionYAML)

	got, err := ExistingPotionIDs(dir)
	require.NoError(t, err)
	assert.True(t, got["potion-1"])
}

func TestExistingPotionIDs_ReadErrorPropagates(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writePotionsFileT(t, dir, ": not: valid: yaml:\n")

	_, err := ExistingPotionIDs(dir)
	require.Error(t, err)
}

func TestExistingPotionIDs_NonMappingRootErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writePotionsFileT(t, dir, "- a\n- b\n")

	_, err := ExistingPotionIDs(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a mapping")
}

func TestWriteProposedPotions_ExistingIDsErrorPropagates(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writePotionsFileT(t, dir, ": not: valid: yaml:\n")
	candidates := []Potion{{ID: "potion-x", ChestID: "runbooks"}}

	_, _, err := WriteProposedPotions(dir, candidates)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "index proposed potions")
}

func TestAppendCandidateToPotionPartition_ReadOrCreateErrorPropagates(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "potions"), []byte("not a dir"), 0o644))
	candidates := []Potion{{ID: "potion-x", ChestID: "runbooks"}}

	_, _, err := WriteProposedPotions(dir, candidates)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "index proposed potions")
}

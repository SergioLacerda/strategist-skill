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

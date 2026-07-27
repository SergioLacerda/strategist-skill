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

// --- query ---

func TestFilterPotions_DefaultExcludesDeprecated(t *testing.T) {
	t.Parallel()
	potions := map[string][]Potion{
		"runbooks": {
			{ID: "potion-1", ChestID: "runbooks", Status: domain.PotionStatusDeprecated},
			{ID: "potion-2", ChestID: "runbooks", Status: domain.PotionStatusProposed},
		},
	}

	got := FilterPotions(potions, PotionFilter{})
	require.Len(t, got, 1)
	assert.Equal(t, "potion-2", got[0].ID)
}

func TestFindPotion(t *testing.T) {
	t.Parallel()
	potions := map[string][]Potion{
		"runbooks": {{ID: "potion-1", ChestID: "runbooks"}},
	}

	got, ok := FindPotion(potions, "potion-1")
	require.True(t, ok)
	assert.Equal(t, "runbooks", got.ChestID)

	_, ok = FindPotion(potions, "missing")
	assert.False(t, ok)
}

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

// --- index scan extension (ask #1 / SQ-001) ---

func TestScanRunbookDirectory_ExtractsWhenToUseFromFirstSection(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sample-issue.md"), []byte(
		"# Runbook: Sample Issue\n\n## Symptom\n\nSomething breaks when X happens.\nIt keeps happening.\n\n## Resolution Steps\n\n1. Do Y.\n"),
		0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Runbooks index\n"), 0o644))

	got, err := ScanRunbookDirectory("runbooks", "T2", dir)
	require.NoError(t, err)
	require.Len(t, got, 1) // README.md excluded
	assert.Equal(t, "potion-sample-issue", got[0].ID)
	assert.Equal(t, "runbooks", got[0].ChestID)
	assert.Equal(t, "docs/runbooks/sample-issue.md", got[0].RunbookRef)
	assert.Equal(t, "Something breaks when X happens. It keeps happening.", got[0].WhenToUse)
	assert.Equal(t, domain.PotionStatusProposed, got[0].Status)
	assert.Equal(t, "T2", got[0].Trust)
}

func TestScanRunbookDirectory_MissingDirIsNotError(t *testing.T) {
	t.Parallel()
	got, err := ScanRunbookDirectory("runbooks", "T2", filepath.Join(t.TempDir(), "does-not-exist"))
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestScanRunbookDirectory_FallsBackToTitleWhenNoSection(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "no-sections.md"), []byte("# Runbook: No Sections\n\nJust a paragraph, no heading.\n"), 0o644))

	got, err := ScanRunbookDirectory("runbooks", "T2", dir)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Runbook: No Sections", got[0].WhenToUse)
}

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

package treasure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

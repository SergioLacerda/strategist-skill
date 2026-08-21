package treasure

import (
	"os"
	"path/filepath"
	"strings"
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

func TestReadRunbookDirEntries_PermissionErrorPropagates(t *testing.T) {
	skipIfPermissionTestUnsupported(t)
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o000))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	_, err := readRunbookDirEntries(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scan runbook directory")
}

func TestIsRunbookCandidateFile_ExcludesDirsAndNonMarkdown(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.md"), nil, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.txt"), nil, 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "c.md"), 0o755))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	got := map[string]bool{}
	for _, e := range entries {
		got[e.Name()] = isRunbookCandidateFile(e)
	}
	assert.True(t, got["a.md"])
	assert.False(t, got["b.txt"])
	assert.False(t, got["c.md"]) // directory, even with .md suffix
}

func TestIsParagraphBoundary_HeadingAlwaysStopsEntirely(t *testing.T) {
	t.Parallel()
	boundary, stop := isParagraphBoundary("## Next Section", true)
	assert.True(t, boundary)
	assert.True(t, stop)

	boundary, stop = isParagraphBoundary("## Next Section", false)
	assert.True(t, boundary)
	assert.True(t, stop)
}

func TestIsParagraphBoundary_BlankLineStopsOnlyWithContent(t *testing.T) {
	t.Parallel()
	boundary, stop := isParagraphBoundary("", true)
	assert.True(t, boundary)
	assert.True(t, stop)

	boundary, stop = isParagraphBoundary("", false)
	assert.True(t, boundary)
	assert.False(t, stop)
}

func TestIsParagraphBoundary_ContentLineIsNotBoundary(t *testing.T) {
	t.Parallel()
	boundary, stop := isParagraphBoundary("some text", false)
	assert.False(t, boundary)
	assert.False(t, stop)
}

func TestRunbookTitleFallback_NoTitleReturnsEmpty(t *testing.T) {
	t.Parallel()
	assert.Empty(t, runbookTitleFallback("no title here\njust text\n"))
}

func TestTruncateRunbookSummary_TruncatesLongString(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("a", 300)
	got := truncateRunbookSummary(long)
	assert.True(t, strings.HasSuffix(got, "…"))
	assert.LessOrEqual(t, len(got), 221+len("…")-1)
}

func TestTruncateRunbookSummary_ShortStringUnchanged(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "short", truncateRunbookSummary("short"))
}

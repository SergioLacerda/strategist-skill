package support

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHelpers(t *testing.T) {
	t.Parallel()
	assert.True(t, ContainsString([]string{"a", "b"}, "b"))
	assert.False(t, ContainsString([]string{"a", "b"}, "c"))
	assert.Equal(t, []string{"discovery", "execution", "refinement"},
		SortedSlotNames(map[string]string{"refinement": "x", "discovery": "y", "execution": "z"}))
	assert.Equal(t, "dev", PackageID("  "))
	assert.Equal(t, "v1.2.3", PackageID(" v1.2.3 "))
}

func TestEnsureGitignoreAppendsOnceAndKeepsExistingLines(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	require.NoError(t, os.WriteFile(path, []byte("node_modules"), 0o644))

	require.NoError(t, EnsureGitignore(dir))
	require.NoError(t, EnsureGitignore(dir))
	require.NoError(t, EnsureGitignoreEntry(dir, UpgradeBackupGitignoreEntry))

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "node_modules\n.strategist/.compiled/\n.strategist/.upgrade-backups/\n", string(raw))
}

func TestEnsureGitignoreReportsAnUnreadableTarget(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".gitignore"), 0o755))
	require.ErrorContains(t, EnsureGitignore(dir), "read .gitignore")
}

func TestCloseGitignoreWrapsTheCloseFailure(t *testing.T) {
	t.Parallel()
	f, err := os.CreateTemp(t.TempDir(), "closed")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.ErrorContains(t, CloseGitignore(f), "close .gitignore")
}

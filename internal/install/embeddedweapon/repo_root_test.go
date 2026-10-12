package embeddedweapon

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepoRootResolvesTheRepositoryFromTheSourceLocation(t *testing.T) {
	t.Parallel()
	root := RepoRoot()
	require.NotEmpty(t, root)
	_, err := os.Stat(filepath.Join(root, "go.mod"))
	require.NoError(t, err, "RepoRoot must point at the directory that holds go.mod")
	assert.True(t, RepositoryRoot(root))
}

func TestRepositoryRootFromWalksUpwardAndStopsAtTheFilesystemRoot(t *testing.T) {
	t.Parallel()
	root := RepoRoot()
	assert.Equal(t, root, RepositoryRootFrom(filepath.Join(root, "internal", "install", "embeddedweapon")))
	assert.Empty(t, RepositoryRootFrom(t.TempDir()))
}

package treasure

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteYAMLNodes_WriteFailure(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "a: 1\n")
	// Directory instead of file path — atomic rename will fail.
	dir := t.TempDir()
	written, err := WriteYAMLNodes(YAMLWrite{Path: dir, Doc: doc})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write")
	assert.Empty(t, written)
}

func TestWriteYAMLNodes_PartialWriteReturnsWrittenPaths(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	goodPath := filepath.Join(dir, "good.yaml")
	doc := mustParseDoc(t, "a: 1\n")

	written, err := WriteYAMLNodes(
		YAMLWrite{Path: goodPath, Doc: doc},
		YAMLWrite{Path: dir, Doc: doc}, // second write fails: dir is not a regular file
	)
	require.Error(t, err)
	assert.Equal(t, []string{goodPath}, written)
}

func TestWriteYAMLNodes_PrepareFailureLeavesNoDestinationMutated(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	goodPath := filepath.Join(dir, "good.yaml")
	require.NoError(t, os.WriteFile(goodPath, []byte("existing: true\n"), 0o644))
	doc := mustParseDoc(t, "a: 1\n")

	// Second write's directory does not exist, so its prepare phase (temp file creation)
	// fails before any rename is attempted for the whole batch.
	badPath := filepath.Join(dir, "missing-subdir", "bad.yaml")

	written, err := WriteYAMLNodes(
		YAMLWrite{Path: goodPath, Doc: doc},
		YAMLWrite{Path: badPath, Doc: doc},
	)
	require.Error(t, err)
	assert.Empty(t, written)

	// goodPath must be untouched: its prepare phase succeeded, but the batch never reached
	// the commit phase because a later write's prepare phase failed.
	raw, readErr := os.ReadFile(goodPath)
	require.NoError(t, readErr)
	assert.Equal(t, "existing: true\n", string(raw))

	// No leftover temp file for goodPath either — prepare-phase cleanup ran for all staged
	// writes, not just the one that failed.
	matches, globErr := filepath.Glob(filepath.Join(dir, ".good.yaml-*.tmp"))
	require.NoError(t, globErr)
	assert.Empty(t, matches)
}

func TestWriteYAMLNodes_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")
	doc := mustParseDoc(t, "a: 1\n")

	written, err := WriteYAMLNodes(YAMLWrite{Path: path, Doc: doc})
	require.NoError(t, err)
	assert.Equal(t, []string{path}, written)
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "a: 1")
}

func TestWriteFileAtomicWithRenameFailurePreservesDestination(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")
	require.NoError(t, os.WriteFile(path, []byte("existing: true\n"), 0o644))

	renameErr := errors.New("rename failed")
	err := writeFileAtomicWithRename(path, []byte("replacement: true\n"), 0o644, func(tmp, dst string) error {
		assert.Equal(t, path, dst)
		_, statErr := os.Stat(tmp)
		require.NoError(t, statErr)
		return renameErr
	})

	require.ErrorIs(t, err, renameErr)
	raw, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	assert.Equal(t, "existing: true\n", string(raw))

	matches, globErr := filepath.Glob(filepath.Join(dir, ".out.yaml-*.tmp"))
	require.NoError(t, globErr)
	assert.Empty(t, matches)
}

func TestWriteFileAtomicUsesRequestedMode(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")

	require.NoError(t, writeFileAtomic(path, []byte("a: 1\n"), 0o600))
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

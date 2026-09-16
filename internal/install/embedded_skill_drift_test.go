package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectoriesHaveDrifted_WantDirMissingErrors(t *testing.T) {
	t.Parallel()
	missing := filepath.Join(t.TempDir(), "no-such-want")

	_, err := directoriesHaveDrifted(missing, t.TempDir())
	require.ErrorContains(t, err, "list expected package")
}

func TestDirectoriesHaveDrifted_GotDirMissingIsDrift(t *testing.T) {
	t.Parallel()
	wantDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(wantDir, "a.md"), []byte("x"), 0o644))
	missingGot := filepath.Join(t.TempDir(), "no-such-got")

	drifted, err := directoriesHaveDrifted(wantDir, missingGot)
	require.NoError(t, err)
	assert.True(t, drifted)
}

func TestDirectoriesHaveDrifted_GotDirUnsupportedEntryErrors(t *testing.T) {
	t.Parallel()
	wantDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(wantDir, "a.md"), []byte("x"), 0o644))
	gotDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(gotDir, "a.md"), []byte("x"), 0o644))
	require.NoError(t, os.Symlink(filepath.Join(gotDir, "a.md"), filepath.Join(gotDir, "link.md")))

	_, err := directoriesHaveDrifted(wantDir, gotDir)
	require.ErrorContains(t, err, "list installed package")
}

func TestDirectoriesHaveDrifted_FileCountMismatchIsDrift(t *testing.T) {
	t.Parallel()
	wantDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(wantDir, "a.md"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(wantDir, "b.md"), []byte("x"), 0o644))
	gotDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(gotDir, "a.md"), []byte("x"), 0o644))

	drifted, err := directoriesHaveDrifted(wantDir, gotDir)
	require.NoError(t, err)
	assert.True(t, drifted)
}

func TestDirectoriesHaveDrifted_ContentMismatchIsDrift(t *testing.T) {
	t.Parallel()
	wantDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(wantDir, "a.md"), []byte("want"), 0o644))
	gotDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(gotDir, "a.md"), []byte("got"), 0o644))

	drifted, err := directoriesHaveDrifted(wantDir, gotDir)
	require.NoError(t, err)
	assert.True(t, drifted)
}

func TestDirectoriesHaveDrifted_IdenticalIsNoDrift(t *testing.T) {
	t.Parallel()
	wantDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(wantDir, "a.md"), []byte("same"), 0o644))
	gotDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(gotDir, "a.md"), []byte("same"), 0o644))

	drifted, err := directoriesHaveDrifted(wantDir, gotDir)
	require.NoError(t, err)
	assert.False(t, drifted)
}

func TestPackageFilesMatch_MissingRelInGot(t *testing.T) {
	t.Parallel()
	wantDir := t.TempDir()
	matching, err := packageFilesMatch(
		map[string]string{"a.md": filepath.Join(wantDir, "a.md")},
		map[string]string{},
	)
	require.NoError(t, err)
	assert.False(t, matching)
}

func TestPackageFilesMatch_PropagatesFilesEqualError(t *testing.T) {
	t.Parallel()
	missing := filepath.Join(t.TempDir(), "missing.md")
	_, err := packageFilesMatch(
		map[string]string{"a.md": missing},
		map[string]string{"a.md": missing},
	)
	require.ErrorContains(t, err, "read expected file")
}

func TestFilesEqual_WantReadErrorPropagates(t *testing.T) {
	t.Parallel()
	missing := filepath.Join(t.TempDir(), "missing.md")
	got := filepath.Join(t.TempDir(), "got.md")
	require.NoError(t, os.WriteFile(got, []byte("x"), 0o644))

	_, err := filesEqual(missing, got)
	require.ErrorContains(t, err, "read expected file")
}

func TestFilesEqual_GotReadErrorPropagates(t *testing.T) {
	t.Parallel()
	want := filepath.Join(t.TempDir(), "want.md")
	require.NoError(t, os.WriteFile(want, []byte("x"), 0o644))
	missingGot := filepath.Join(t.TempDir(), "missing.md")

	_, err := filesEqual(want, missingGot)
	require.ErrorContains(t, err, "read installed file")
}

func TestRegularFilePath_WalkErrorPropagates(t *testing.T) {
	t.Parallel()
	_, _, err := regularFilePath(t.TempDir(), "/whatever", nil, assert.AnError)
	require.ErrorIs(t, err, assert.AnError)
}

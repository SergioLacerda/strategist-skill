package runtimefs_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafeJoin_ContainedRelativePath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	got, err := runtimefs.SafeJoin(root, filepath.Join("a", "b.txt"))
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "a", "b.txt"), got)
}

func TestSafeJoin_RootItself(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	got, err := runtimefs.SafeJoin(root, ".")
	require.NoError(t, err)
	assert.Equal(t, filepath.Clean(root), got)
}

func TestSafeJoin_RejectsDotDotEscape(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	_, err := runtimefs.SafeJoin(root, filepath.Join("..", "..", "etc", "passwd"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "escapes root")
}

func TestSafeJoin_RejectsDotDotEscapeAfterDescending(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	// Starts inside root, then climbs past it — filepath.Join/Clean collapses
	// this before SafeJoin ever sees a literal "..", so the containment
	// check (not a naive ".." string search) is what has to catch it.
	_, err := runtimefs.SafeJoin(root, filepath.Join("a", "..", "..", "escaped"))
	require.Error(t, err)
}

func TestSafeJoin_AbsoluteRelIsTreatedAsRelative(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	// filepath.Join treats an absolute second argument as just another
	// path segment, not as an override of root — confirm SafeJoin keeps
	// that (safe) behavior rather than accidentally honoring it as absolute.
	got, err := runtimefs.SafeJoin(root, filepath.Join(string(filepath.Separator), "etc", "passwd"))
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(got))
	assert.Contains(t, got, root)
}

func TestSafeJoin_EmptyRoot(t *testing.T) {
	t.Parallel()

	_, err := runtimefs.SafeJoin("", "a")
	require.Error(t, err)
}

func TestSafeJoinExisting_ContainedPathPassesThrough(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "f.txt"), []byte("x"), 0o644))

	got, err := runtimefs.SafeJoinExisting(root, "f.txt")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "f.txt"), got)
}

func TestSafeJoinExisting_NonExistentPathFallsBackToSafeJoin(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	got, err := runtimefs.SafeJoinExisting(root, "does-not-exist.txt")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "does-not-exist.txt"), got)
}

func TestSafeJoinExisting_RejectsPlainDotDotEscape(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	_, err := runtimefs.SafeJoinExisting(root, filepath.Join("..", "escaped"))
	require.Error(t, err)
}

func TestSafeJoinExisting_DetectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on Windows")
	}
	t.Parallel()

	root := t.TempDir()
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("s"), 0o644))

	// "link" lives inside root but points outside it — SafeJoin alone (no
	// filesystem access) would accept "link/whatever" as contained, since
	// the literal path string never leaves root. SafeJoinExisting must
	// catch this once the symlink itself resolves.
	link := filepath.Join(root, "link")
	require.NoError(t, os.Symlink(outside, link))

	plain, err := runtimefs.SafeJoin(root, "link")
	require.NoError(t, err, "SafeJoin has no filesystem access and cannot see the symlink target")
	assert.Equal(t, link, plain)

	_, err = runtimefs.SafeJoinExisting(root, "link")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "symlink")
}

func TestSafeJoinExisting_SymlinkStayingInsideRootIsAllowed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on Windows")
	}
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "real"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "real", "f.txt"), []byte("x"), 0o644))
	require.NoError(t, os.Symlink(filepath.Join(root, "real"), filepath.Join(root, "link")))

	_, err := runtimefs.SafeJoinExisting(root, filepath.Join("link", "f.txt"))
	require.NoError(t, err)
}

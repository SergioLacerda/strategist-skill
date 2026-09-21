package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveInstallTargetReturnsAbsolutePaths(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	explicit, err := resolveInstallTarget("sub", false)
	require.NoError(t, err)
	require.True(t, filepath.IsAbs(explicit))
	require.Equal(t, filepath.Join(dir, "sub"), explicit)

	fallback, err := resolveInstallTarget("", false)
	require.NoError(t, err)
	require.True(t, filepath.IsAbs(fallback))
}

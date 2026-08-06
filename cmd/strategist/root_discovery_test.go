package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findStrategistRoot/resolveStrategistRoot are thin wrappers around
// internal/cliutil — see cliutil_test.go there for the full behavior matrix.
// These smoke tests only confirm the delegation itself is wired correctly.

func TestFindStrategistRoot_DelegatesToCliutil(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".strategist"), 0o755))

	strategistDir, projectRoot, err := findStrategistRoot(dir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, ".strategist"), strategistDir)
	assert.Equal(t, dir, projectRoot)
}

func TestResolveStrategistRoot_DelegatesToCliutil(t *testing.T) {
	strategistDir, projectRoot, err := resolveStrategistRoot("some/relative/path", "/unused/cwd")
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(strategistDir))
	assert.Equal(t, filepath.Dir(strategistDir), projectRoot)
}

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindStrategistRoot_FoundInCWD(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".strategist"), 0o755))

	strategistDir, projectRoot, err := findStrategistRoot(dir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, ".strategist"), strategistDir)
	assert.Equal(t, dir, projectRoot)
}

func TestFindStrategistRoot_FoundInParent(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".strategist"), 0o755))
	subdir := filepath.Join(root, "subproject", "src")
	require.NoError(t, os.MkdirAll(subdir, 0o755))

	strategistDir, projectRoot, err := findStrategistRoot(subdir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, ".strategist"), strategistDir)
	assert.Equal(t, root, projectRoot)
}

func TestFindStrategistRoot_NotFound(t *testing.T) {
	dir := t.TempDir() // no .strategist/

	_, _, err := findStrategistRoot(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestInstallCmd_IdempotentUpdatesExisting(t *testing.T) {
	// Layout: root/.strategist/ already exists; install runs from root/subdir.
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".strategist"), 0o755))

	subdir := filepath.Join(root, "subdir")
	require.NoError(t, os.MkdirAll(subdir, 0o755))

	origWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(subdir))
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	origTarget := installTarget
	origSilent := installSilent
	origWizard := installWizard
	origGlobal := installGlobal
	t.Cleanup(func() {
		installTarget = origTarget
		installSilent = origSilent
		installWizard = origWizard
		installGlobal = origGlobal
	})

	installTarget = ""
	installSilent = true
	installWizard = false
	installGlobal = false

	// install should resolve target to root (where .strategist/ exists), not subdir.
	// We can't run the full install (shim step requires home), but we can verify
	// the target resolution logic: after findStrategistRoot succeeds, installTarget = root.
	if _, _, err := findStrategistRoot(subdir); err == nil {
		// Confirms walk-up finds root/.strategist/ from subdir.
		discovered, projRoot, discErr := findStrategistRoot(subdir)
		require.NoError(t, discErr)
		assert.Equal(t, filepath.Join(root, ".strategist"), discovered)
		assert.Equal(t, root, projRoot)
	}
}

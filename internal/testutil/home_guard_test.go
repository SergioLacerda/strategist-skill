package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangedUserShimsDetectsCreationAndRewrite(t *testing.T) {
	home := t.TempDir()
	claude := filepath.Join(home, userShimRelPaths[0])
	before := snapshotUserShims(home)
	assert.Empty(t, changedUserShims(home, before), "an untouched home reports nothing")

	require.NoError(t, os.MkdirAll(filepath.Dir(claude), 0o750))
	require.NoError(t, os.WriteFile(claude, []byte("skill_root: /tmp/leak\n"), 0o600))
	assert.Equal(t, []string{userShimRelPaths[0]}, changedUserShims(home, before), "a created shim is a violation")

	rewritten := snapshotUserShims(home)
	require.NoError(t, os.WriteFile(claude, []byte("skill_root: /tmp/other\n"), 0o600))
	assert.Equal(t, []string{userShimRelPaths[0]}, changedUserShims(home, rewritten), "a rewritten shim is a violation")
}

func TestRunWithHomeGuard_ExecutionPaths(t *testing.T) {
	assert.Equal(t, 0, RunWithHomeGuard(nil))

	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("USERPROFILE", fakeHome)

	// Clean run
	res := runWithHomeGuardRunner(func() int { return 0 })
	assert.Equal(t, 0, res)

	// Violation run (file created in fake Home during test execution)
	resViolation := runWithHomeGuardRunner(func() int {
		claude := filepath.Join(fakeHome, userShimRelPaths[0])
		require.NoError(t, os.MkdirAll(filepath.Dir(claude), 0o750))
		require.NoError(t, os.WriteFile(claude, []byte("leak"), 0o600))
		return 0
	})
	assert.Equal(t, 1, resViolation)

	// Failure run with non-zero exit code
	resFail := runWithHomeGuardRunner(func() int { return 42 })
	assert.Equal(t, 42, resFail)

	// Error resolving user home
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	resError := runWithHomeGuardRunner(func() int { return 99 })
	assert.Equal(t, 99, resError)
}

func TestRealUserHome(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("USERPROFILE", fakeHome)

	home, err := realUserHome()
	require.NoError(t, err)
	assert.Equal(t, fakeHome, home)

	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	_, err = realUserHome()
	assert.Error(t, err)
}

package main

// Tests for warnIfConfigModified (root.go): it must resolve the actual
// .strategist runtime root via findStrategistRoot rather than a path
// hardcoded relative to cwd, so the warning behaves the same from the
// project root and from a subdirectory.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/integrity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func chdirForTest(t *testing.T, dir string) {
	t.Helper()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
}

func TestWarnIfConfigModified_QuietWhenNoRuntimeRoot(t *testing.T) {
	chdirForTest(t, t.TempDir())

	out := captureStderr(t, warnIfConfigModified)
	assert.Empty(t, out)
}

func TestWarnIfConfigModified_QuietWhenNoLockYet(t *testing.T) {
	root := t.TempDir()
	strategistDir := filepath.Join(root, ".strategist")
	require.NoError(t, os.MkdirAll(strategistDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(strategistDir, "active.yaml"), []byte("mode: epic\n"), 0o644))
	chdirForTest(t, root)

	out := captureStderr(t, warnIfConfigModified)
	assert.Empty(t, out, "first install has no lock yet — must stay quiet")
}

func TestWarnIfConfigModified_SilentWhenUnmodified(t *testing.T) {
	root := t.TempDir()
	strategistDir := filepath.Join(root, ".strategist")
	require.NoError(t, os.MkdirAll(strategistDir, 0o755))
	activePath := filepath.Join(strategistDir, "active.yaml")
	require.NoError(t, os.WriteFile(activePath, []byte("mode: epic\n"), 0o644))
	require.NoError(t, integrity.WriteLock(activePath, filepath.Join(strategistDir, ".config.lock")))
	chdirForTest(t, root)

	out := captureStderr(t, warnIfConfigModified)
	assert.Empty(t, out)
}

func TestWarnIfConfigModified_WarnsFromSubdirectory(t *testing.T) {
	root := t.TempDir()
	strategistDir := filepath.Join(root, ".strategist")
	require.NoError(t, os.MkdirAll(strategistDir, 0o755))
	activePath := filepath.Join(strategistDir, "active.yaml")
	require.NoError(t, os.WriteFile(activePath, []byte("mode: epic\n"), 0o644))
	require.NoError(t, integrity.WriteLock(activePath, filepath.Join(strategistDir, ".config.lock")))

	// External edit after sealing — no CLI re-seal happened.
	require.NoError(t, os.WriteFile(activePath, []byte("mode: full\n"), 0o644))

	sub := filepath.Join(root, "nested", "deeper")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	chdirForTest(t, sub)

	out := captureStderr(t, warnIfConfigModified)
	assert.Contains(t, out, "modified outside the CLI")
	assert.Contains(t, out, "reason=")
}

func TestWarnIfConfigModified_ConfigMissingWarns(t *testing.T) {
	root := t.TempDir()
	strategistDir := filepath.Join(root, ".strategist")
	require.NoError(t, os.MkdirAll(strategistDir, 0o755))
	activePath := filepath.Join(strategistDir, "active.yaml")
	require.NoError(t, os.WriteFile(activePath, []byte("mode: epic\n"), 0o644))
	require.NoError(t, integrity.WriteLock(activePath, filepath.Join(strategistDir, ".config.lock")))
	require.NoError(t, os.Remove(activePath))
	chdirForTest(t, root)

	out := captureStderr(t, warnIfConfigModified)
	assert.Contains(t, out, "active.yaml missing after lock")
}

func TestWarnIfConfigModified_PathMismatchWarns(t *testing.T) {
	root := t.TempDir()
	strategistDir := filepath.Join(root, ".strategist")
	require.NoError(t, os.MkdirAll(strategistDir, 0o755))
	activePath := filepath.Join(strategistDir, "active.yaml")
	require.NoError(t, os.WriteFile(activePath, []byte("mode: epic\n"), 0o644))
	lockPath := filepath.Join(strategistDir, ".config.lock")
	require.NoError(t, integrity.WriteLock(activePath, lockPath))

	// Hand-craft the lock to record a different sealed path than the one
	// that will actually be checked, without moving any real directories.
	raw, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	var lock map[string]any
	require.NoError(t, json.Unmarshal(raw, &lock))
	lock["path"] = filepath.Join(root, "elsewhere", "active.yaml")
	rewritten, err := json.Marshal(lock)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(lockPath, rewritten, 0o644))

	chdirForTest(t, root)

	out := captureStderr(t, warnIfConfigModified)
	assert.Contains(t, out, "lock path mismatch")
}

func TestWarnIfConfigModified_CorruptLockWarnsWithoutBlocking(t *testing.T) {
	root := t.TempDir()
	strategistDir := filepath.Join(root, ".strategist")
	require.NoError(t, os.MkdirAll(strategistDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(strategistDir, "active.yaml"), []byte("mode: epic\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(strategistDir, ".config.lock"), []byte("not json"), 0o644))
	chdirForTest(t, root)

	out := captureStderr(t, warnIfConfigModified)
	assert.Contains(t, out, "lock is corrupt")
}

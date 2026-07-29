package integrity_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/integrity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteLock_and_CheckUnmodified(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "active.yaml")
	lockPath := filepath.Join(dir, ".config.lock")

	if err := os.WriteFile(configPath, []byte("mode: epic\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := integrity.WriteLock(configPath, lockPath); err != nil {
		t.Fatalf("WriteLock: %v", err)
	}

	modified, err := integrity.IsModified(configPath, lockPath)
	if err != nil {
		t.Fatalf("IsModified: %v", err)
	}
	if modified {
		t.Error("expected IsModified=false immediately after WriteLock")
	}
}

func TestIsModified_detects_external_change(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "active.yaml")
	lockPath := filepath.Join(dir, ".config.lock")

	if err := os.WriteFile(configPath, []byte("mode: epic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := integrity.WriteLock(configPath, lockPath); err != nil {
		t.Fatal(err)
	}

	// Simulate external edit: advance mtime by 1 second via Chtimes
	future := time.Now().Add(time.Second)
	if err := os.Chtimes(configPath, future, future); err != nil {
		t.Fatal(err)
	}

	modified, err := integrity.IsModified(configPath, lockPath)
	if err != nil {
		t.Fatalf("IsModified: %v", err)
	}
	if !modified {
		t.Error("expected IsModified=true after external mtime change")
	}
}

func TestIsModified_no_lock_file(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "active.yaml")
	lockPath := filepath.Join(dir, ".config.lock")

	if err := os.WriteFile(configPath, []byte("mode: epic\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	modified, err := integrity.IsModified(configPath, lockPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if modified {
		t.Error("expected IsModified=false when no lock file exists")
	}
}

func TestWriteLock_StatError(t *testing.T) {
	dir := t.TempDir()
	err := integrity.WriteLock(filepath.Join(dir, "missing.yaml"), filepath.Join(dir, ".config.lock"))
	if err == nil {
		t.Fatal("expected error for missing config path")
	}
}

func TestWriteLock_WriteFileError(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "active.yaml")
	if err := os.WriteFile(configPath, []byte("mode: epic\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// lockPath's parent directory does not exist, so WriteFile fails.
	err := integrity.WriteLock(configPath, filepath.Join(dir, "missing-dir", ".config.lock"))
	if err == nil {
		t.Fatal("expected error when lock directory does not exist")
	}
}

func TestIsModified_CorruptLockFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "active.yaml")
	lockPath := filepath.Join(dir, ".config.lock")

	if err := os.WriteFile(configPath, []byte("mode: epic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockPath, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := integrity.IsModified(configPath, lockPath)
	if err == nil {
		t.Fatal("expected error for corrupt lock file")
	}
}

func TestIsModified_ConfigRemovedAfterLock(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "active.yaml")
	lockPath := filepath.Join(dir, ".config.lock")

	if err := os.WriteFile(configPath, []byte("mode: epic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := integrity.WriteLock(configPath, lockPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}

	// Config missing after a sealed lock is a stronger drift signal than
	// "unmodified" — flag it so callers don't silently trust an absent config.
	modified, err := integrity.IsModified(configPath, lockPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !modified {
		t.Error("expected IsModified=true when config file no longer exists after a lock was sealed")
	}

	result, err := integrity.Check(configPath, lockPath)
	require.NoError(t, err)
	assert.True(t, result.Modified)
	assert.Equal(t, integrity.ReasonConfigMissing, result.Reason)
}

func TestIsModified_ReadLockError(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "active.yaml")
	lockPath := filepath.Join(dir, ".config.lock")

	if err := os.WriteFile(configPath, []byte("mode: epic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// lockPath is a directory, so ReadFile fails with a non-ErrNotExist error.
	if err := os.Mkdir(lockPath, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := integrity.IsModified(configPath, lockPath)
	if err == nil {
		t.Fatal("expected error when lock path is a directory")
	}
}

func TestIsModified_StatConfigError(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "active.yaml")
	lockPath := filepath.Join(dir, ".config.lock")

	if err := os.WriteFile(configPath, []byte("mode: epic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := integrity.WriteLock(configPath, lockPath); err != nil {
		t.Fatal(err)
	}

	// Replace configPath's parent-relative access with a path that traverses a
	// non-directory component, producing an ENOTDIR stat error (not ErrNotExist).
	badConfigPath := filepath.Join(configPath, "impossible-child")

	_, err := integrity.IsModified(badConfigPath, lockPath)
	if err == nil {
		t.Fatal("expected stat error for path with non-directory component")
	}
}

func TestCheck_UnmodifiedReportsReasonAndPaths(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "active.yaml")
	lockPath := filepath.Join(dir, ".config.lock")

	require.NoError(t, os.WriteFile(configPath, []byte("mode: epic\n"), 0o644))
	require.NoError(t, integrity.WriteLock(configPath, lockPath))

	result, err := integrity.Check(configPath, lockPath)
	require.NoError(t, err)
	assert.False(t, result.Modified)
	assert.Equal(t, integrity.ReasonUnmodified, result.Reason)
	assert.Equal(t, configPath, result.ConfigPath)
	assert.Equal(t, lockPath, result.LockPath)
}

func TestCheck_NoLockReportsLockMissing(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "active.yaml")
	lockPath := filepath.Join(dir, ".config.lock")
	require.NoError(t, os.WriteFile(configPath, []byte("mode: epic\n"), 0o644))

	result, err := integrity.Check(configPath, lockPath)
	require.NoError(t, err)
	assert.False(t, result.Modified)
	assert.Equal(t, integrity.ReasonLockMissing, result.Reason)
}

// TestCheck_HashMismatchDetectedEvenWithSameMtime guards against an attacker (or a
// tool) restoring mtime after editing content — mtime alone would miss this, but
// the sealed sha256 catches the content change regardless of mtime.
func TestCheck_HashMismatchDetectedEvenWithSameMtime(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "active.yaml")
	lockPath := filepath.Join(dir, ".config.lock")

	require.NoError(t, os.WriteFile(configPath, []byte("mode: epic\n"), 0o644))
	require.NoError(t, integrity.WriteLock(configPath, lockPath))

	info, err := os.Stat(configPath)
	require.NoError(t, err)
	originalMtime := info.ModTime()

	require.NoError(t, os.WriteFile(configPath, []byte("mode: full\n"), 0o644))
	require.NoError(t, os.Chtimes(configPath, originalMtime, originalMtime))

	result, err := integrity.Check(configPath, lockPath)
	require.NoError(t, err)
	assert.True(t, result.Modified)
	assert.Equal(t, integrity.ReasonHashMismatch, result.Reason)
}

// TestCheck_SizeMismatchWithoutHash exercises the size-based fallback for a lock
// that carries size but no hash — the new schema always writes both together, but
// Check must still honor a size-only lock on its own terms.
func TestCheck_SizeMismatchWithoutHash(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "active.yaml")
	lockPath := filepath.Join(dir, ".config.lock")

	require.NoError(t, os.WriteFile(configPath, []byte("mode: epic\n"), 0o644))
	info, err := os.Stat(configPath)
	require.NoError(t, err)

	partial := struct {
		Schema string    `json:"schema,omitempty"`
		Path   string    `json:"path"`
		Mtime  time.Time `json:"mtime"`
		Size   int64     `json:"size,omitempty"`
	}{Schema: "strategist-config-lock/1.0", Path: configPath, Mtime: info.ModTime().UTC(), Size: info.Size() + 1}
	data, err := json.Marshal(partial)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(lockPath, data, 0o600))

	result, err := integrity.Check(configPath, lockPath)
	require.NoError(t, err)
	assert.True(t, result.Modified)
	assert.Equal(t, integrity.ReasonSizeMismatch, result.Reason)
}

// TestCheck_LegacyLockAcceptedAsUnmodified verifies pre-hardening locks (only
// mtime + path) still work and are classified via Detail, not rejected as corrupt.
func TestCheck_LegacyLockAcceptedAsUnmodified(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "active.yaml")
	lockPath := filepath.Join(dir, ".config.lock")

	require.NoError(t, os.WriteFile(configPath, []byte("mode: epic\n"), 0o644))
	info, err := os.Stat(configPath)
	require.NoError(t, err)

	legacy := struct {
		Path  string    `json:"path"`
		Mtime time.Time `json:"mtime"`
	}{Path: configPath, Mtime: info.ModTime().UTC()}
	data, err := json.Marshal(legacy)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(lockPath, data, 0o600))

	result, err := integrity.Check(configPath, lockPath)
	require.NoError(t, err)
	assert.False(t, result.Modified)
	assert.Equal(t, integrity.ReasonLegacyLock, result.Reason)
	assert.NotEmpty(t, result.Detail)
}

// TestCheck_LegacyLockDetectsMtimeChange verifies legacy (mtime-only) locks still
// catch drift, they just can't distinguish content change from a bare touch.
func TestCheck_LegacyLockDetectsMtimeChange(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "active.yaml")
	lockPath := filepath.Join(dir, ".config.lock")

	require.NoError(t, os.WriteFile(configPath, []byte("mode: epic\n"), 0o644))
	info, err := os.Stat(configPath)
	require.NoError(t, err)

	legacy := struct {
		Path  string    `json:"path"`
		Mtime time.Time `json:"mtime"`
	}{Path: configPath, Mtime: info.ModTime().UTC()}
	data, err := json.Marshal(legacy)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(lockPath, data, 0o600))

	future := time.Now().Add(time.Second)
	require.NoError(t, os.Chtimes(configPath, future, future))

	result, err := integrity.Check(configPath, lockPath)
	require.NoError(t, err)
	assert.True(t, result.Modified)
	assert.Equal(t, integrity.ReasonMTimeMismatch, result.Reason)
}

// TestCheck_PathMismatch guards against a stale lock sealed for a different
// config path being reused accidentally (e.g. after a treasure-chest path change).
func TestCheck_PathMismatch(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "active.yaml")
	otherPath := filepath.Join(dir, "other.yaml")
	lockPath := filepath.Join(dir, ".config.lock")

	require.NoError(t, os.WriteFile(configPath, []byte("mode: epic\n"), 0o644))
	require.NoError(t, os.WriteFile(otherPath, []byte("mode: epic\n"), 0o644))
	require.NoError(t, integrity.WriteLock(otherPath, lockPath))

	result, err := integrity.Check(configPath, lockPath)
	require.NoError(t, err)
	assert.True(t, result.Modified)
	assert.Equal(t, integrity.ReasonPathMismatch, result.Reason)
	assert.NotEmpty(t, result.Detail)
}

func TestWriteLock_SetsLockFilePermissionsAndLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "active.yaml")
	lockPath := filepath.Join(dir, ".config.lock")
	require.NoError(t, os.WriteFile(configPath, []byte("mode: epic\n"), 0o644))
	require.NoError(t, integrity.WriteLock(configPath, lockPath))

	info, err := os.Stat(lockPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		assert.False(t, strings.HasPrefix(e.Name(), ".config.lock.tmp-"), "temp lock file leaked: %s", e.Name())
	}
}

func TestWriteLock_AtomicWriteFailureLeavesNoTempFileAndNoPartialLock(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "active.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("mode: epic\n"), 0o644))

	// lockPath's parent directory does not exist, so the temp-file create fails
	// before any rename — no partial lock should ever appear at the target path.
	lockPath := filepath.Join(dir, "missing-dir", ".config.lock")
	err := integrity.WriteLock(configPath, lockPath)
	require.Error(t, err)

	_, statErr := os.Stat(lockPath)
	assert.True(t, os.IsNotExist(statErr))
}

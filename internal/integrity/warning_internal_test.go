package integrity

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHashFile_DoesNotExist is a white-box test for hashFile's own
// "does not exist" branch — reachable directly (path was never Stat-gated
// by a caller), unlike via WriteLock/Check, which always Stat the config
// path before ever calling hashFile.
func TestHashFile_DoesNotExist(t *testing.T) {
	_, err := hashFile(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected an error for a nonexistent path, got nil")
	}
}

// faultyLockTempFile is a fault-injecting fake of lockTempFile:
// writeLockFileAtomically never gets a handle to the real *os.File it
// creates, so Write/Close/Chmod failures on that specific file have no
// black-box trigger — this fake lets tests force each branch directly.
type faultyLockTempFile struct {
	name     string
	writeErr error
	closeErr error
}

func (f *faultyLockTempFile) Write(p []byte) (int, error) {
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	return len(p), nil
}

func (f *faultyLockTempFile) Close() error { return f.closeErr }

func (f *faultyLockTempFile) Name() string { return f.name }

func withFaultyLockTempFile(t *testing.T, fake *faultyLockTempFile) {
	t.Helper()
	origCreate := createLockTempFile
	t.Cleanup(func() { createLockTempFile = origCreate })
	createLockTempFile = func(_, _ string) (lockTempFile, error) {
		return fake, nil
	}
}

func TestWriteLockFileAtomically_WriteError(t *testing.T) {
	dir := t.TempDir()
	tmpPath := filepath.Join(dir, "fake.tmp")
	if err := os.WriteFile(tmpPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	withFaultyLockTempFile(t, &faultyLockTempFile{name: tmpPath, writeErr: errors.New("boom")})

	err := writeLockFileAtomically(filepath.Join(dir, ".config.lock"), []byte("data"), 0o600)
	if err == nil || !strings.Contains(err.Error(), "write temp lock file") {
		t.Fatalf("expected a write-temp-lock-file error, got %v", err)
	}
}

func TestWriteLockFileAtomically_WriteErrorAndCloseErrorJoined(t *testing.T) {
	dir := t.TempDir()
	tmpPath := filepath.Join(dir, "fake.tmp")
	if err := os.WriteFile(tmpPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	withFaultyLockTempFile(t, &faultyLockTempFile{
		name:     tmpPath,
		writeErr: errors.New("write boom"),
		closeErr: errors.New("close boom"),
	})

	err := writeLockFileAtomically(filepath.Join(dir, ".config.lock"), []byte("data"), 0o600)
	if err == nil || !strings.Contains(err.Error(), "write boom") || !strings.Contains(err.Error(), "close boom") {
		t.Fatalf("expected a joined write+close error, got %v", err)
	}
}

func TestWriteLockFileAtomically_CloseError(t *testing.T) {
	dir := t.TempDir()
	tmpPath := filepath.Join(dir, "fake.tmp")
	if err := os.WriteFile(tmpPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	withFaultyLockTempFile(t, &faultyLockTempFile{name: tmpPath, closeErr: errors.New("close boom")})

	err := writeLockFileAtomically(filepath.Join(dir, ".config.lock"), []byte("data"), 0o600)
	if err == nil || !strings.Contains(err.Error(), "close temp lock file") {
		t.Fatalf("expected a close-temp-lock-file error, got %v", err)
	}
}

func TestWriteLockFileAtomically_ChmodError(t *testing.T) {
	dir := t.TempDir()
	tmpPath := filepath.Join(dir, "fake.tmp")
	if err := os.WriteFile(tmpPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	withFaultyLockTempFile(t, &faultyLockTempFile{name: tmpPath})

	origChmod := chmodLockTempFile
	t.Cleanup(func() { chmodLockTempFile = origChmod })
	chmodLockTempFile = func(_ string, _ os.FileMode) error {
		return errors.New("chmod boom")
	}

	err := writeLockFileAtomically(filepath.Join(dir, ".config.lock"), []byte("data"), 0o600)
	if err == nil || !strings.Contains(err.Error(), "chmod temp lock file") {
		t.Fatalf("expected a chmod-temp-lock-file error, got %v", err)
	}
}

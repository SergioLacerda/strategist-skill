package integrity_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/integrity"
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

	modified, err := integrity.IsModified(configPath, lockPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if modified {
		t.Error("expected IsModified=false when config file no longer exists")
	}
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

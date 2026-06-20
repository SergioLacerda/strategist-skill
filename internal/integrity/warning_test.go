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

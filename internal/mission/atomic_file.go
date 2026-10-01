package mission

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteFileAtomic writes data to path through a same-directory temporary file
// and a rename, so readers never observe a partial file. The parent directory
// must exist.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".atomic-*.tmp")
	if err != nil {
		return fmt.Errorf("atomic write: create temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }() //nolint:errcheck // best-effort cleanup of an uncommitted temporary file.
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close() //nolint:errcheck // best-effort cleanup before returning the chmod error.
		return fmt.Errorf("atomic write: set permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close() //nolint:errcheck // best-effort cleanup before returning the write error.
		return fmt.Errorf("atomic write: write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("atomic write: close: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("atomic write: persist: %w", err)
	}
	return nil
}

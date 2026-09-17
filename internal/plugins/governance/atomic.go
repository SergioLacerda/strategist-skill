package governance

import (
	"fmt"
	"os"
	"path/filepath"
)

func atomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create governance directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".governance-*")
	if err != nil {
		return fmt.Errorf("create governance temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) //nolint:errcheck // cleanup after the atomic rename attempt
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close() //nolint:errcheck // preserve the original chmod error
		return fmt.Errorf("chmod governance temporary file: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close() //nolint:errcheck // preserve the original write error
		return fmt.Errorf("write governance temporary file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close governance temporary file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename governance temporary file: %w", err)
	}
	return nil
}

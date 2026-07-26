// Package runtimefs provides Strategist-agnostic filesystem primitives shared by runtime modules.
package runtimefs

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Exists reports whether path exists, regardless of file type.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ReadSHA256 returns the sha256 hex digest for path.
func ReadSHA256(path string) (hash string, exists bool, err error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: caller owns path trust boundary
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read sha256: %w", err)
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum), true, nil
}

// WriteFile writes data to path after creating the parent directory.
func WriteFile(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir parent %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, perm); err != nil { //nolint:gosec // G306: caller controls desired file mode
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

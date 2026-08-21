// Package runtimefs provides Strategist-agnostic filesystem primitives shared by runtime modules.
package runtimefs

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// gzTempFile is the subset of *os.File that WriteGzJSON needs. createGzTempFile
// exists only so tests can substitute a fault-injecting fake for the file's
// own Close call — a plain *os.File.Close() on a regular local file has no
// realistic black-box trigger (see runtimefs_test.go).
type gzTempFile interface {
	io.Writer
	Close() error
}

var createGzTempFile = func(path string) (gzTempFile, error) {
	return os.Create(path) //nolint:gosec // G304: caller owns path trust boundary
}

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

// WriteGzJSON atomically writes v as gzip-compressed JSON.
func WriteGzJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir parent %s: %w", filepath.Dir(path), err)
	}

	tmp := path + ".tmp"
	f, err := createGzTempFile(tmp)
	if err != nil {
		return fmt.Errorf("create tmp %s: %w", tmp, err)
	}

	cleanup := func() {
		_ = os.Remove(tmp) //nolint:errcheck // best-effort cleanup after failed atomic write
	}
	gz := gzip.NewWriter(f)
	if err := json.NewEncoder(gz).Encode(v); err != nil {
		_ = gz.Close() //nolint:errcheck // close before cleanup so Windows can remove tmp
		_ = f.Close()  //nolint:errcheck // best-effort after encode failure
		cleanup()
		return fmt.Errorf("json encode: %w", err)
	}
	if err := gz.Close(); err != nil {
		cleanup()
		return fmt.Errorf("gzip close: %w", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return fmt.Errorf("file close: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		cleanup()
		return fmt.Errorf("rename %s: %w", path, err)
	}
	return nil
}

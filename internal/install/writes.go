package install

import (
	"fmt"
	"os"
	"path/filepath"
)

// atomicWriteFile creates path's parent directory if needed, then writes data
// to path atomically: it writes to a temporary file in the same directory and
// renames it into place, so a crash or interruption never leaves a partially
// written critical file (active.yaml, install manifest, provider manifests,
// shim files, knowledge/treasure-chest manifests).
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir parent %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup; no-op after a successful rename

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close() //nolint:errcheck,gosec // already failing; cleanup handled by deferred Remove
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename into place: %w", err)
	}
	return nil
}

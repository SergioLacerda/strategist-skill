// Package filesystem provides Strategist-agnostic filesystem primitives shared
// by runtime and other internal modules.
//
// # gosec G304 policy
//
// G304 fires on any variable path passed to a file-read/open/create call —
// a purely syntactic check that cannot tell a genuinely attacker-influenced
// path from one that is safe by construction. Across this codebase, G304
// sites fall into three categories:
//
//  1. root + compile-time-constant filename (e.g. filepath.Join(strategistDir,
//     "active.yaml")). No injection surface: the joined component can never
//     vary at runtime, so SafeJoin adds no protection a nolint comment
//     didn't already correctly claim. Left as-is with a nolint comment.
//  2. root + filesystem-glob-discovered path (e.g. jewels/*.yaml expansion).
//     Safe by construction: glob results can never resolve outside the
//     directory they were expanded from. Left as-is with a nolint comment.
//  3. root + a component read from external/untrusted content (a workspace
//     config value, a chest entry, CLI/wizard input echoed into a path).
//     This is the real G304 risk category — these call sites should route
//     through SafeJoin or SafeJoinExisting (see internal/cli/cliutil.ResolveActiveBasePath
//     and internal/install's ensureGitignore for worked examples). Any new
//     call site that builds a path from data the process did not itself
//     enumerate belongs in this category.
package filesystem

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
// realistic black-box trigger (see filesystem_test.go).
type gzTempFile interface {
	io.Writer
	Close() error
}

var createGzTempFile = func(path string) (gzTempFile, error) {
	return os.Create(path) //nolint:gosec // G304: caller owns path trust boundary
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

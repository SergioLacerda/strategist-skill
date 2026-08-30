// Package runtimefs provides Strategist-agnostic filesystem primitives shared
// by runtime modules.
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
//     through SafeJoin or SafeJoinExisting (see internal/cliutil.ResolveActiveBasePath
//     and internal/install's ensureGitignore for worked examples). Any new
//     call site that builds a path from data the process did not itself
//     enumerate belongs in this category.
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
	"strings"
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

// SafeJoin joins root and rel, normalizes the result, and returns an error
// if it would escape root (e.g. via a ".." component or an absolute rel that
// cleans outside root). It never touches the filesystem — no existence
// check, no symlink resolution — so it cannot catch a symlink inside root
// that points outside it; use SafeJoinExisting for that. Callers that build
// a path from caller-supplied input (a CLI flag, a config value, a path
// read from a workspace file) should route it through SafeJoin or
// SafeJoinExisting instead of a raw filepath.Join before passing it to an
// os.* file operation (addresses gosec G304 — see
// .analysis/refined/20260830-pending-v3-disposition E20).
func SafeJoin(root, rel string) (string, error) {
	if root == "" {
		return "", errors.New("safejoin: empty root")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("safejoin: resolve root %q: %w", root, err)
	}
	absRoot = filepath.Clean(absRoot)
	joined := filepath.Clean(filepath.Join(absRoot, rel))
	if joined != absRoot && !strings.HasPrefix(joined, absRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("safejoin: %q escapes root %q", rel, root)
	}
	return joined, nil
}

// SafeJoinExisting behaves like SafeJoin, then additionally resolves
// symlinks on whichever side already exists on disk and re-checks
// containment against the resolved paths. This closes the gap SafeJoin
// alone leaves open: a path component inside root that is itself a symlink
// pointing outside root. If root or the joined path does not exist yet (a
// common, legitimate case — e.g. writing a new file), symlink resolution is
// skipped for that side and the plain SafeJoin result is returned; this
// does not remove TOCTOU risk (the target could be replaced by a symlink
// between this check and the caller's actual file operation) — callers
// needing that stronger guarantee would need directory-fd-relative
// operations, which this package does not provide.
func SafeJoinExisting(root, rel string) (string, error) {
	joined, err := SafeJoin(root, rel)
	if err != nil {
		return "", err
	}
	resolvedRoot, rootErr := filepath.EvalSymlinks(root)
	resolvedJoined, joinedErr := filepath.EvalSymlinks(joined)
	if rootErr != nil || joinedErr != nil {
		return joined, nil
	}
	resolvedRoot = filepath.Clean(resolvedRoot)
	resolvedJoined = filepath.Clean(resolvedJoined)
	if resolvedJoined != resolvedRoot && !strings.HasPrefix(resolvedJoined, resolvedRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("safejoin: %q resolves via symlink outside root %q", rel, root)
	}
	return joined, nil
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

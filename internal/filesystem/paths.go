package filesystem

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PathResolver implements the domain runtime path port with local filesystem
// canonicalization and identity checks.
type PathResolver struct{}

// Canonical returns the normalized, symlink-resolved form of path when it can
// be resolved, falling back to the absolute or cleaned input otherwise.
func (PathResolver) Canonical(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return abs
}

// SameDirectory reports whether left and right identify the same filesystem
// object.
func (PathResolver) SameDirectory(left, right string) bool {
	infoLeft, errLeft := os.Stat(left)
	infoRight, errRight := os.Stat(right)
	if errLeft != nil || errRight != nil {
		return false
	}
	return os.SameFile(infoLeft, infoRight)
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

// IsMissingPath reports whether err means that path is absent. On Windows,
// opening a child below a regular file can also surface as ERROR_PATH_NOT_FOUND
// and therefore match os.ErrNotExist. Walk to the first existing ancestor so
// that a missing path remains optional while a file used as a directory is an
// actual error.
func IsMissingPath(path string, err error) bool {
	if !errors.Is(err, os.ErrNotExist) {
		return false
	}
	// Some platforms report directory operations on an existing regular file
	// with a not-found-style error. The target itself takes precedence over the
	// parent walk: an existing file is not an absent optional path.
	if _, statErr := os.Stat(path); statErr == nil {
		return false
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return false
	}
	return existingAncestorIsDirectory(filepath.Dir(path))
}

func existingAncestorIsDirectory(path string) bool {
	for current := path; ; current = filepath.Dir(current) {
		info, statErr := os.Stat(current)
		if statErr == nil {
			return info.IsDir()
		}
		if !errors.Is(statErr, os.ErrNotExist) {
			return false
		}
		parent := filepath.Dir(current)
		if parent == current {
			return true
		}
	}
}

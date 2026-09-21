package runtimepayload

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func notRegular(name string) error {
	return fmt.Errorf("%w: %q is not a regular file", ErrUnsafeArchive, name)
}

// closeInto records a Close failure unless an earlier error already won.
func closeInto(c io.Closer, what string, err *error) {
	if closeErr := c.Close(); closeErr != nil && *err == nil {
		*err = fmt.Errorf("close %s: %w", what, closeErr)
	}
}

// writeEntry writes one regular file under root after the entry name has been
// stripped and proven to stay inside root.
func writeEntry(root, name string, strip int, mode fs.FileMode, r io.Reader, b *budget) error {
	rel, ok, err := entryPath(name, strip)
	if err != nil || !ok {
		return err
	}
	if err := b.file(); err != nil {
		return err
	}
	target, err := containedTarget(root, rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(target), err)
	}
	return copyToFile(target, entryPerm(mode), r, b)
}

// containedTarget joins rel under root and re-checks the result: entryPath
// already rejects escapes, and this second, path-level check guarantees the
// write target cannot leave root whatever the entry name was.
func containedTarget(root, rel string) (string, error) {
	cleanRoot := filepath.Clean(root)
	target := filepath.Join(cleanRoot, filepath.FromSlash(rel))
	if !strings.HasPrefix(target, cleanRoot+string(os.PathSeparator)) {
		return "", fmt.Errorf("%w: %q escapes %s", ErrUnsafeArchive, rel, cleanRoot)
	}
	return target, nil
}

func entryPerm(mode fs.FileMode) fs.FileMode {
	if mode&0o111 != 0 {
		return 0o755
	}
	return 0o644
}

func copyToFile(target string, perm fs.FileMode, r io.Reader, b *budget) error {
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm) //nolint:gosec // target proven inside root by containedTarget
	if err != nil {
		return fmt.Errorf("create %s: %w", target, err)
	}
	n, copyErr := io.Copy(out, io.LimitReader(r, maxExtractedBytes-b.bytes+1))
	closeErr := out.Close()
	b.bytes += n
	if b.bytes > maxExtractedBytes {
		return fmt.Errorf("%w: extracted size exceeds limit", ErrUnsafeArchive)
	}
	if copyErr != nil {
		return fmt.Errorf("write %s: %w", target, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close %s: %w", target, closeErr)
	}
	return nil
}

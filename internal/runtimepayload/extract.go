package runtimepayload

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func unsafeArchive(err error) error { return fmt.Errorf("%w: %v", ErrUnsafeArchive, err) }

func notRegular(name string) error {
	return fmt.Errorf("%w: %q is not a regular file", ErrUnsafeArchive, name)
}

// closeInto records a Close failure unless an earlier error already won.
func closeInto(c io.Closer, what string, err *error) {
	if closeErr := c.Close(); closeErr != nil && *err == nil {
		*err = fmt.Errorf("close %s: %w", what, closeErr)
	}
}

func extractTarGz(data []byte, root string, strip int, b *budget) (err error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return unsafeArchive(err)
	}
	defer closeInto(gz, "gzip stream", &err)
	return extractTarEntries(tar.NewReader(gz), root, strip, b)
}

func extractTarEntries(tr *tar.Reader, root string, strip int, b *budget) error {
	for {
		hdr, done, err := nextTarHeader(tr)
		if done || err != nil {
			return err
		}
		if err := extractTarEntry(hdr, tr, root, strip, b); err != nil {
			return err
		}
	}
}

// nextTarHeader returns the next header; done is true at a clean end of archive.
func nextTarHeader(tr *tar.Reader) (hdr *tar.Header, done bool, err error) {
	hdr, err = tr.Next()
	if errors.Is(err, io.EOF) {
		return nil, true, nil
	}
	if err != nil {
		return nil, false, unsafeArchive(err)
	}
	return hdr, false, nil
}

func extractTarEntry(hdr *tar.Header, r io.Reader, root string, strip int, b *budget) error {
	switch hdr.Typeflag {
	case tar.TypeDir:
		return nil
	case tar.TypeReg:
		return writeEntry(root, hdr.Name, strip, hdr.FileInfo().Mode(), r, b)
	default:
		return notRegular(hdr.Name)
	}
}

func extractZip(data []byte, root string, strip int, b *budget) error {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return unsafeArchive(err)
	}
	for _, f := range zr.File {
		if err := extractZipFile(f, root, strip, b); err != nil {
			return err
		}
	}
	return nil
}

func extractZipFile(f *zip.File, root string, strip int, b *budget) (err error) {
	if f.FileInfo().IsDir() {
		return nil
	}
	if !f.Mode().IsRegular() {
		return notRegular(f.Name)
	}
	rc, err := f.Open()
	if err != nil {
		return unsafeArchive(err)
	}
	defer closeInto(rc, f.Name, &err)
	return writeEntry(root, f.Name, strip, f.Mode(), rc, b)
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
	target := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(target), err)
	}
	return copyToFile(target, entryPerm(mode), r, b)
}

func entryPerm(mode fs.FileMode) fs.FileMode {
	if mode&0o111 != 0 {
		return 0o755
	}
	return 0o644
}

func copyToFile(target string, perm fs.FileMode, r io.Reader, b *budget) error {
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm) //nolint:gosec // target proven inside root by entryPath
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

package acquisition

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxArchiveBytes = 64 * 1024 * 1024
	maxArchiveFiles = 4096
)

func extractArchive(root string, data []byte) error {
	if int64(len(data)) > maxArchiveBytes {
		return acquisitionError("archive_too_large", fmt.Sprintf("archive exceeds %d bytes", maxArchiveBytes))
	}
	reader := tar.NewReader(bytes.NewReader(data))
	seen := map[string]struct{}{}
	files := 0
	for {
		done, err := extractNextArchiveEntry(root, reader, seen, &files)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}
}

func extractNextArchiveEntry(root string, reader *tar.Reader, seen map[string]struct{}, files *int) (bool, error) {
	header, err := reader.Next()
	if err == io.EOF {
		return true, nil
	}
	if err != nil {
		return false, acquisitionError("archive_invalid", err.Error())
	}
	if isPAXHeader(header) {
		return false, nil
	}
	name, err := nextArchivePath(header, seen, files)
	if err != nil {
		return false, err
	}
	return false, extractArchiveEntry(root, reader, header, name)
}

func isPAXHeader(header *tar.Header) bool {
	return header.Typeflag == tar.TypeXHeader || header.Typeflag == tar.TypeXGlobalHeader
}

func nextArchivePath(header *tar.Header, seen map[string]struct{}, files *int) (string, error) {
	if *files >= maxArchiveFiles {
		return "", acquisitionError("archive_too_many_files", fmt.Sprintf("archive exceeds %d entries", maxArchiveFiles))
	}
	name, err := cleanRelativeArchivePath(header.Name)
	if err != nil {
		return "", err
	}
	if _, exists := seen[name]; exists {
		return "", acquisitionError("archive_duplicate_entry", name)
	}
	seen[name] = struct{}{}
	*files++
	return name, nil
}

func extractArchiveEntry(root string, reader *tar.Reader, header *tar.Header, name string) error {
	target := filepath.Join(root, filepath.FromSlash(name))
	if err := ensureContained(root, target); err != nil {
		return err
	}
	switch header.Typeflag {
	case tar.TypeDir:
		return makeArchiveDirectory(target)
	case tar.TypeReg:
		return extractRegularFile(target, reader, header.Size, name)
	case tar.TypeSymlink, tar.TypeLink:
		return acquisitionError("unsafe_entry", "links are not allowed: "+name)
	default:
		return acquisitionError("unsafe_entry", "unsupported archive entry: "+name)
	}
}

func makeArchiveDirectory(target string) error {
	if err := os.MkdirAll(target, 0o700); err != nil {
		return acquisitionError("archive_write_failed", err.Error())
	}
	return nil
}

func extractRegularFile(target string, reader *tar.Reader, size int64, name string) error {
	if err := validateArchiveFileSize(size, name); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return acquisitionError("archive_write_failed", err.Error())
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) //nolint:gosec // target is contained by a fresh quarantine
	if err != nil {
		return acquisitionError("archive_write_failed", err.Error())
	}
	return copyArchiveFile(file, reader, size)
}

func validateArchiveFileSize(size int64, name string) error {
	if size < 0 || size > maxArchiveBytes {
		return acquisitionError("archive_file_too_large", name)
	}
	return nil
}

func copyArchiveFile(file *os.File, reader *tar.Reader, size int64) error {
	_, copyErr := io.CopyN(file, reader, size)
	closeErr := file.Close()
	if copyErr != nil && (copyErr != io.EOF || size != 0) {
		return acquisitionError("archive_invalid", copyErr.Error())
	}
	if closeErr != nil {
		return acquisitionError("archive_write_failed", closeErr.Error())
	}
	return nil
}

func ensureContained(root, target string) error {
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return acquisitionError("archive_path_invalid", "archive entry escapes quarantine")
	}
	return nil
}

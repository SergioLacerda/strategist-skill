package connectors

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

type fileEntry struct {
	relPath string
	content []byte
}

func readPackageFile(root, path string, entry fs.DirEntry, walkErr error, allowRuntime bool) (*fileEntry, int64, error) {
	if walkErr != nil {
		return nil, 0, walkErr
	}
	if entry.IsDir() {
		return nil, 0, nil
	}
	rel, err := packageRelPath(root, path)
	if err != nil {
		return nil, 0, err
	}
	data, err := readBoundedPackageData(path, rel, allowRuntime)
	if err != nil {
		return nil, 0, err
	}
	return &fileEntry{relPath: filepath.ToSlash(rel), content: data}, int64(len(data)), nil
}

func packageRelPath(root, path string) (string, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", fmt.Errorf("relative path for %q under %q: %w", path, root, err)
	}
	if len(rel) > domain.MaxPluginPathLength {
		return "", fmt.Errorf("path %q exceeds %d characters", rel, domain.MaxPluginPathLength)
	}
	return rel, nil
}

// readBoundedPackageData reads a package file, enforcing the manifest byte
// limit (the larger embedded-runtime limit applies only under embeddedRuntimeDir).
func readBoundedPackageData(path, rel string, allowRuntime bool) ([]byte, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: root is an operator-declared ingestion source, not untrusted request input
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}
	limit := domain.MaxPluginManifestBytes
	if allowRuntime && strings.HasPrefix(filepath.ToSlash(rel), embeddedRuntimeDir) {
		limit = MaxEmbeddedRuntimeFileBytes
	}
	if len(data) > limit {
		return nil, fmt.Errorf("file %q exceeds %d bytes", rel, limit)
	}
	return data, nil
}

// Package embed provides the embedded default Strategist skill files and the extractor to install them.
package embed

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed all:defaults
var defaultsFS embed.FS

// Extractor implements domain.FileExtractor using the embedded defaults.
type Extractor struct{}

// Extract copies all embedded defaults into targetDir, preserving the directory
// structure but stripping the leading "defaults/" path prefix.
func (e Extractor) Extract(targetDir string) error {
	return extractFS(defaultsFS, "defaults", targetDir)
}

// extractFS copies files from src under root into targetDir.
// Separated from Extract to allow injecting arbitrary fs.FS in tests.
func extractFS(src fs.FS, root, targetDir string) error {
	if err := fs.WalkDir(src, root, makeWalkFn(src, root, targetDir)); err != nil {
		return fmt.Errorf("embed: %w", err)
	}
	return nil
}

func makeWalkFn(src fs.FS, root, targetDir string) fs.WalkDirFunc {
	return func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("embed: walk %s: %w", path, err)
		}
		rel := strings.TrimPrefix(path, root+"/")
		if rel == root || rel == "" {
			return nil
		}
		dst := filepath.Join(targetDir, rel)
		if d.IsDir() {
			if mkErr := os.MkdirAll(dst, 0o755); mkErr != nil {
				return fmt.Errorf("embed: mkdir %s: %w", dst, mkErr)
			}
			return nil
		}
		return writeEmbedFile(src, path, dst)
	}
}

func writeEmbedFile(src fs.FS, path, dst string) error {
	data, readErr := fs.ReadFile(src, path)
	if readErr != nil {
		return fmt.Errorf("embed: read %s: %w", path, readErr)
	}
	if mkErr := os.MkdirAll(filepath.Dir(dst), 0o755); mkErr != nil {
		return fmt.Errorf("embed: mkdir parent %s: %w", filepath.Dir(dst), mkErr)
	}
	if writeErr := os.WriteFile(dst, data, 0o644); writeErr != nil {
		return fmt.Errorf("embed: write %s: %w", dst, writeErr)
	}
	return nil
}

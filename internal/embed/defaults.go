// Package embed provides the embedded default Strategist skill files and the extractor to install them.
package embed

import (
	"crypto/sha256"
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
//
// When force is false (merge mode), files that already exist on disk and whose
// content differs from the embedded default are skipped — the user's customizations
// are preserved. Files that match the embedded default are overwritten (idempotent).
// When force is true, all files are overwritten unconditionally.
func (e Extractor) Extract(targetDir string, force bool) error {
	return extractFS(defaultsFS, "defaults", targetDir, force)
}

// extractFS copies files from src under root into targetDir.
// Separated from Extract to allow injecting arbitrary fs.FS in tests.
func extractFS(src fs.FS, root, targetDir string, force bool) error {
	if err := fs.WalkDir(src, root, makeWalkFn(src, root, targetDir, force)); err != nil {
		return fmt.Errorf("embed: %w", err)
	}
	return nil
}

func makeWalkFn(src fs.FS, root, targetDir string, force bool) fs.WalkDirFunc {
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
		return writeEmbedFile(src, path, dst, force)
	}
}

// writeEmbedFile writes embedded file src/path to dst.
// In merge mode (!force), if dst already exists and its content differs from the
// embedded version, the file is skipped to preserve user customizations.
func writeEmbedFile(src fs.FS, path, dst string, force bool) error {
	data, readErr := fs.ReadFile(src, path)
	if readErr != nil {
		return fmt.Errorf("embed: read %s: %w", path, readErr)
	}
	if mkErr := os.MkdirAll(filepath.Dir(dst), 0o755); mkErr != nil {
		return fmt.Errorf("embed: mkdir parent %s: %w", filepath.Dir(dst), mkErr)
	}
	if !force {
		if userModified(dst, data) {
			return nil // preserve user's version
		}
	}
	if writeErr := os.WriteFile(dst, data, 0o644); writeErr != nil {
		return fmt.Errorf("embed: write %s: %w", dst, writeErr)
	}
	return nil
}

// userModified reports true when dst exists on disk AND its content differs
// from the embedded bytes — meaning the user has customized the file.
func userModified(dst string, embedded []byte) bool {
	existing, err := os.ReadFile(dst)
	if err != nil {
		return false // file doesn't exist — not user-modified
	}
	return sha256.Sum256(existing) != sha256.Sum256(embedded)
}

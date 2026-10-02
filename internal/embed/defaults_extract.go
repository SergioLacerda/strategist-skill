package embed

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
)

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
		rel, ok, err := embeddedRelPath(path, root, err)
		if err != nil || !ok {
			return err
		}
		dst := filepath.Join(targetDir, rel)
		if d.IsDir() {
			return ensureEmbedDir(dst)
		}
		return writeEmbedFile(src, path, dst, force)
	}
}

func embeddedRelPath(path, root string, walkErr error) (string, bool, error) {
	if walkErr != nil {
		return "", false, fmt.Errorf("embed: walk %s: %w", path, walkErr)
	}
	rel := strings.TrimPrefix(path, root+"/")
	if rel == root || rel == "" {
		return "", false, nil
	}
	return rel, true, nil
}

func ensureEmbedDir(dst string) error {
	if mkErr := os.MkdirAll(dst, 0o755); mkErr != nil {
		return fmt.Errorf("embed: mkdir %s: %w", dst, mkErr)
	}
	return nil
}

type extractMode bool

const (
	extractMerge     extractMode = false
	extractOverwrite extractMode = true
)

func extractModeFor(force bool) extractMode {
	if force {
		return extractOverwrite
	}
	return extractMerge
}

// writeEmbedFile writes embedded file src/path to dst.
// In merge mode, if dst already exists and its content differs from the
// embedded version, the file is skipped to preserve user customizations.
func writeEmbedFile(src fs.FS, path, dst string, force bool) error {
	mode := extractModeFor(force)
	data, readErr := fs.ReadFile(src, path)
	if readErr != nil {
		return fmt.Errorf("embed: read %s: %w", path, readErr)
	}
	if mode == extractMerge {
		if userModified(dst, data) {
			return nil // preserve user's version
		}
	}
	if writeErr := runtimefs.WriteFile(dst, data, 0o644); writeErr != nil {
		return fmt.Errorf("embed: write %s: %w", dst, writeErr)
	}
	return nil
}

// userModified reports true when dst exists on disk AND its content differs
// from the embedded bytes — meaning the user has customized the file.
func userModified(dst string, embedded []byte) bool {
	embeddedHash := fmt.Sprintf("%x", sha256Bytes(embedded))
	existingHash, exists, err := runtimefs.ReadSHA256(dst)
	if err != nil || !exists {
		return false // file doesn't exist or cannot be read — not user-modified
	}
	return existingHash != embeddedHash
}

func sha256Bytes(data []byte) [32]byte {
	return sha256.Sum256(data)
}

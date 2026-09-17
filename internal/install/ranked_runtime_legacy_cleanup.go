package install

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
)

func removeLegacyNestedOpenSpecRoot(root string) error {
	legacyRoot := filepath.Join(root, "openspec")
	if !runtimefs.Exists(filepath.Join(legacyRoot, "config.yaml")) {
		return nil
	}
	if err := validateLegacyNestedOpenSpecRoot(legacyRoot); err != nil {
		return err
	}
	if err := os.RemoveAll(legacyRoot); err != nil {
		return fmt.Errorf("remove empty legacy OpenSpec root: %w", err)
	}
	return nil
}

func validateLegacyNestedOpenSpecRoot(legacyRoot string) error {
	unexpected, err := findUnexpectedLegacyPath(legacyRoot, legacyOpenSpecAllowedPaths())
	if err != nil {
		return fmt.Errorf("inspect legacy OpenSpec root: %w", err)
	}
	if unexpected != "" {
		return fmt.Errorf("legacy OpenSpec root contains user content at %s; migrate it before reinstalling", unexpected)
	}
	return nil
}

func legacyOpenSpecAllowedPaths() map[string]struct{} {
	return map[string]struct{}{
		"config.yaml":              {},
		"changes":                  {},
		"changes/archive":          {},
		"changes/archive/.gitkeep": {},
		"specs":                    {},
		"specs/.gitkeep":           {},
	}
}

func findUnexpectedLegacyPath(root string, allowed map[string]struct{}) (string, error) {
	var unexpected string
	err := filepath.WalkDir(root, func(path string, _ os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		candidate, err := legacyPathIfUnexpected(root, path, allowed)
		if err != nil {
			return err
		}
		if candidate != "" {
			unexpected = candidate
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return unexpected, fmt.Errorf("walk legacy OpenSpec root: %w", err)
	}
	return unexpected, nil
}

func legacyPathIfUnexpected(root, path string, allowed map[string]struct{}) (string, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", fmt.Errorf("compute legacy OpenSpec relative path: %w", err)
	}
	if rel == "." {
		return "", nil
	}
	rel = filepath.ToSlash(rel)
	if _, ok := allowed[rel]; ok {
		return "", nil
	}
	return rel, nil
}

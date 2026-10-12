package rankedbinding

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/filesystem"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
)

// RemoveLegacyNestedOpenSpecRoot removes the legacy nested openspec directory under root when it contains only
// allowed paths.
func RemoveLegacyNestedOpenSpecRoot(root string) error {
	legacyRoot := filepath.Join(root, "openspec")
	if !filesystem.Exists(filepath.Join(legacyRoot, names.RuntimeConfigName)) {
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
	unexpected, err := FindUnexpectedLegacyPath(legacyRoot, LegacyOpenSpecAllowedPaths())
	if err != nil {
		return fmt.Errorf("inspect legacy OpenSpec root: %w", err)
	}
	if unexpected != "" {
		return fmt.Errorf("legacy OpenSpec root contains user content at %s; migrate it before reinstalling", unexpected)
	}
	return nil
}

// LegacyOpenSpecAllowedPaths returns the paths a legacy nested OpenSpec root may contain.
func LegacyOpenSpecAllowedPaths() map[string]struct{} {
	return map[string]struct{}{
		names.RuntimeConfigName:    {},
		"changes":                  {},
		"changes/archive":          {},
		"changes/archive/.gitkeep": {},
		"specs":                    {},
		"specs/.gitkeep":           {},
	}
}

// FindUnexpectedLegacyPath returns the first path under root that a legacy OpenSpec root must not contain.
func FindUnexpectedLegacyPath(root string, allowed map[string]struct{}) (string, error) {
	var unexpected string
	err := filepath.WalkDir(root, func(path string, _ os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		candidate, err := LegacyPathIfUnexpected(root, path, allowed)
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

// LegacyPathIfUnexpected returns path relative to root, slash-separated, when it is not in allowed; it returns an
// empty string for root itself and for allowed paths.
func LegacyPathIfUnexpected(root, path string, allowed map[string]struct{}) (string, error) {
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

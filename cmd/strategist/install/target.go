package install

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveTarget derives the target without depending on package-main state.
func ResolveTarget(explicit string, global bool, discover func(string) (string, string, error)) (string, error) {
	if explicit != "" {
		return absoluteTarget(explicit)
	}
	if global {
		return globalTarget()
	}
	return localTarget(discover)
}

func globalTarget() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("install: resolve home dir: %w", err)
	}
	return home, nil
}

func localTarget(discover func(string) (string, string, error)) (string, error) {
	cwd, err := os.Getwd()
	if err == nil && discover != nil {
		if _, projectRoot, err := discover(cwd); err == nil {
			return projectRoot, nil
		}
	}
	return absoluteTarget(".")
}
func absoluteTarget(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("install: resolve absolute target %q: %w", path, err)
	}
	return absolute, nil
}

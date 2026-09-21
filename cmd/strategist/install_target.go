package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// resolveInstallTarget returns the effective install target path.
// For global installs, returns the user home dir.
// For local installs with no explicit target, walks up from CWD to find an
// existing .strategist/ and updates in-place; falls back to "." otherwise.
func resolveInstallTarget(explicit string, global bool) (string, error) {
	if explicit != "" {
		return absoluteInstallTarget(explicit)
	}
	if global {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("install: resolve home dir: %w", err)
		}
		return home, nil
	}
	if cwd, err := os.Getwd(); err == nil {
		if _, projRoot, discErr := findStrategistRoot(cwd); discErr == nil {
			return projRoot, nil
		}
	}
	return absoluteInstallTarget(".")
}

func absoluteInstallTarget(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("install: resolve absolute target %q: %w", path, err)
	}
	return absolute, nil
}

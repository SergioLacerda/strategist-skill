package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const strategistDirName = ".strategist"
const rootDiscoveryMaxLevels = 5

// findStrategistRoot walks up from startDir looking for a .strategist/ directory.
// Returns (strategistDir, projectRoot, nil) on success.
// strategistDir is the full path to .strategist/.
// projectRoot is its parent (the project root).
// Returns an error if no .strategist/ is found within rootDiscoveryMaxLevels levels.
func findStrategistRoot(startDir string) (strategistDir, projectRoot string, err error) {
	dir := startDir
	for i := 0; i < rootDiscoveryMaxLevels; i++ {
		candidate := filepath.Join(dir, strategistDirName)
		if stat, statErr := os.Stat(candidate); statErr == nil && stat.IsDir() {
			return candidate, dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		dir = parent
	}
	return "", "", fmt.Errorf("%s not found within %d levels of %s — run: strategist install", strategistDirName, rootDiscoveryMaxLevels, startDir)
}

// resolveStrategistRoot returns the runtime root to use for a command.
// If explicit is non-empty, it is used directly (projectRoot = its parent).
// Otherwise, findStrategistRoot is called from cwd.
func resolveStrategistRoot(explicit, cwd string) (strategistDir, projectRoot string, err error) {
	if explicit != "" {
		abs, absErr := filepath.Abs(explicit)
		if absErr != nil {
			return "", "", fmt.Errorf("resolve root: %w", absErr)
		}
		return abs, filepath.Dir(abs), nil
	}
	return findStrategistRoot(cwd)
}

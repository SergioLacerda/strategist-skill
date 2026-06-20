// Package integrity provides config integrity helpers for the Strategist CLI.
package integrity

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

type configLock struct {
	Mtime time.Time `json:"mtime"`
	Path  string    `json:"path"`
}

// WriteLock records the current mtime of configPath into lockPath.
// Call this immediately after writing active.yaml during install.
func WriteLock(configPath, lockPath string) error {
	info, err := os.Stat(configPath)
	if err != nil {
		return fmt.Errorf("integrity: stat config: %w", err)
	}
	lock := configLock{Mtime: info.ModTime().UTC(), Path: configPath}
	data, err := json.Marshal(lock)
	if err != nil {
		return fmt.Errorf("integrity: marshal lock: %w", err)
	}
	if err := os.WriteFile(lockPath, data, 0o600); err != nil {
		return fmt.Errorf("integrity: write lock: %w", err)
	}
	return nil
}

// IsModified reports whether configPath has been modified since the last WriteLock.
// Returns false (not an error) when no lock file exists — first install scenario.
func IsModified(configPath, lockPath string) (bool, error) {
	data, err := os.ReadFile(lockPath) //nolint:gosec // G304: lockPath is a known internal path
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("integrity: read lock: %w", err)
	}

	var lock configLock
	if err := json.Unmarshal(data, &lock); err != nil {
		return false, fmt.Errorf("integrity: parse lock: %w", err)
	}

	info, err := os.Stat(configPath)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("integrity: stat config: %w", err)
	}

	return !info.ModTime().UTC().Equal(lock.Mtime), nil
}

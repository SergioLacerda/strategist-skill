package integrity

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
)

// statConfig, lockPathMismatch, pathMismatchResult, compareConfigState,
// configDriftReason, lockHashMismatch, lockMTimeMismatch, readLock, hashFile,
// and normalizePath are Check's internal drift-comparison helpers, split out
// of warning.go to keep that file under the repo's file-size budget.

func statConfig(configPath string) (os.FileInfo, bool, error) {
	info, err := os.Stat(configPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, true, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("integrity: stat config: %w", err)
	}
	return info, false, nil
}

func lockPathMismatch(lock configLock, configPath string) bool {
	return lock.Path != "" && normalizePath(lock.Path) != normalizePath(configPath)
}

func pathMismatchResult(result Result, lock configLock, configPath string) Result {
	result.Modified = true
	result.Reason = ReasonPathMismatch
	result.Detail = fmt.Sprintf("lock was sealed for %q, checked path is %q", lock.Path, configPath)
	return result
}

func compareConfigState(result Result, configPath string, lock configLock, info os.FileInfo) (Result, error) {
	reason, modified, err := configDriftReason(configPath, lock, info)
	if err != nil {
		return Result{}, err
	}
	if modified {
		result.Modified = true
		result.Reason = reason
		return result, nil
	}
	if lock.Schema == "" {
		result.Reason = ReasonLegacyLock
		result.Detail = "legacy lock schema (mtime-only fingerprint); re-run install to upgrade"
		return result, nil
	}
	result.Reason = ReasonUnmodified
	return result, nil
}

func configDriftReason(configPath string, lock configLock, info os.FileInfo) (Reason, bool, error) {
	if hashMismatch, err := lockHashMismatch(configPath, lock); err != nil || hashMismatch {
		return ReasonHashMismatch, hashMismatch, err
	}
	if lock.Size > 0 && info.Size() != lock.Size {
		return ReasonSizeMismatch, true, nil
	}
	if lockMTimeMismatch(lock, info) {
		return ReasonMTimeMismatch, true, nil
	}
	return "", false, nil
}

func lockHashMismatch(configPath string, lock configLock) (bool, error) {
	if lock.SHA256 == "" {
		return false, nil
	}
	sum, err := hashFile(configPath)
	if err != nil {
		return false, fmt.Errorf("integrity: hash config: %w", err)
	}
	return sum != lock.SHA256, nil
}

func lockMTimeMismatch(lock configLock, info os.FileInfo) bool {
	if lock.MtimeNS != 0 {
		return info.ModTime().UnixNano() != lock.MtimeNS
	}
	return !info.ModTime().UTC().Equal(lock.Mtime)
}

func readLock(lockPath string) (configLock, bool, error) {
	data, err := os.ReadFile(lockPath) //nolint:gosec // G304: lockPath is a known internal path
	if errors.Is(err, os.ErrNotExist) {
		return configLock{}, false, nil
	}
	if err != nil {
		return configLock{}, false, fmt.Errorf("integrity: read lock: %w", err)
	}
	var lock configLock
	if err := json.Unmarshal(data, &lock); err != nil {
		return configLock{}, false, fmt.Errorf("integrity: parse lock: %w", err)
	}
	return lock, true, nil
}

// hashFile reuses internal/runtimefs's fingerprint helper — the same one
// internal/stale uses for artifact hashing — so both drift-detection modules
// agree on one hash format ("sha256:<hex>") without merging their packages.
func hashFile(path string) (string, error) {
	hash, exists, err := runtimefs.ReadSHA256(path)
	if err != nil {
		return "", fmt.Errorf("integrity: read config hash: %w", err)
	}
	if !exists {
		return "", fmt.Errorf("integrity: %s does not exist", path)
	}
	return "sha256:" + hash, nil
}

// normalizePath resolves path to an absolute, cleaned form so that equivalent
// paths reached from different working directories compare equal.
func normalizePath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(abs)
}

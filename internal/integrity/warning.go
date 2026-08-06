// Package integrity provides config drift-detection helpers for the Strategist CLI.
//
// This is operational drift detection, not tamper-proof security: a local actor
// who can edit both active.yaml and .config.lock can reseal malicious changes.
// The goal is to catch accidental or manual edits made outside the CLI.
package integrity

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
)

// lockSchemaV1 identifies the current lock JSON shape. Locks written before this
// field existed ("legacy" locks) contain only mtime and path.
const lockSchemaV1 = "strategist-config-lock/1.0"

// Reason explains why a Result reports (or does not report) drift.
type Reason string

// Reason values describe config lock comparison outcomes.
const (
	ReasonUnmodified    Reason = "unmodified"
	ReasonLockMissing   Reason = "lock_missing"
	ReasonConfigMissing Reason = "config_missing"
	ReasonMTimeMismatch Reason = "mtime_mismatch"
	ReasonHashMismatch  Reason = "hash_mismatch"
	ReasonSizeMismatch  Reason = "size_mismatch"
	ReasonPathMismatch  Reason = "path_mismatch"
	ReasonLegacyLock    Reason = "legacy_lock"
)

// Result is the structured outcome of a Check.
type Result struct {
	Modified   bool   `json:"modified"`
	Reason     Reason `json:"reason"`
	ConfigPath string `json:"config_path"`
	LockPath   string `json:"lock_path"`
	Detail     string `json:"detail,omitempty"`
}

// configLock is the on-disk lock shape. Fields beyond Mtime/Path were added in
// lockSchemaV1; a lock with an empty Schema is a legacy mtime-only lock.
type configLock struct {
	Schema  string    `json:"schema,omitempty"`
	Path    string    `json:"path"`
	Mtime   time.Time `json:"mtime"`
	MtimeNS int64     `json:"mtime_ns,omitempty"`
	Size    int64     `json:"size,omitempty"`
	SHA256  string    `json:"sha256,omitempty"`
}

// WriteLock records a fingerprint of configPath into lockPath.
// Call this immediately after writing active.yaml during install or after any
// other CLI-trusted mutation of the config file.
func WriteLock(configPath, lockPath string) error {
	info, err := os.Stat(configPath)
	if err != nil {
		return fmt.Errorf("integrity: stat config: %w", err)
	}
	sum, err := hashFile(configPath)
	if err != nil {
		return fmt.Errorf("integrity: hash config: %w", err)
	}
	lock := configLock{
		Schema:  lockSchemaV1,
		Path:    normalizePath(configPath),
		Mtime:   info.ModTime().UTC(),
		MtimeNS: info.ModTime().UnixNano(),
		Size:    info.Size(),
		SHA256:  sum,
	}
	data, err := json.Marshal(lock)
	if err != nil {
		return fmt.Errorf("integrity: marshal lock: %w", err)
	}
	if err := writeLockFileAtomically(lockPath, data, 0o600); err != nil {
		return fmt.Errorf("integrity: write lock: %w", err)
	}
	return nil
}

// IsModified reports whether configPath has been modified since the last WriteLock.
// It is a compatibility wrapper around Check; prefer Check for diagnostics.
func IsModified(configPath, lockPath string) (bool, error) {
	result, err := Check(configPath, lockPath)
	if err != nil {
		return false, err
	}
	return result.Modified, nil
}

// Check compares the current state of configPath against the fingerprint sealed
// in lockPath and returns a structured Result explaining the outcome.
func Check(configPath, lockPath string) (Result, error) {
	result := Result{ConfigPath: configPath, LockPath: lockPath}

	lock, found, err := readLock(lockPath)
	if err != nil {
		return Result{}, err
	}
	if !found {
		result.Reason = ReasonLockMissing
		return result, nil
	}

	info, missing, err := statConfig(configPath)
	if missing {
		result.Modified = true
		result.Reason = ReasonConfigMissing
		return result, nil
	}
	if err != nil {
		return Result{}, err
	}

	if lockPathMismatch(lock, configPath) {
		return pathMismatchResult(result, lock, configPath), nil
	}

	return compareConfigState(result, configPath, lock, info)
}

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

// lockTempFile is the subset of *os.File that writeLockFileAtomically needs.
// createLockTempFile and chmodLockTempFile below exist only so tests can
// substitute a fault-injecting fake for the Write/Close/Chmod calls on a
// temp file the caller otherwise never gets a handle to — see
// warning_internal_test.go.
type lockTempFile interface {
	io.Writer
	Close() error
	Name() string
}

var createLockTempFile = func(dir, pattern string) (lockTempFile, error) {
	return os.CreateTemp(dir, pattern)
}

var chmodLockTempFile = os.Chmod

// writeLockFileAtomically writes data to a sibling temp file in lockPath's
// directory, then renames it into place, so a process interruption never
// leaves a partially written lock file. It does not create missing parent
// directories: the lock's directory (e.g. .strategist/) is expected to exist.
func writeLockFileAtomically(lockPath string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(lockPath)
	tmp, err := createLockTempFile(dir, ".config.lock.tmp-*")
	if err != nil {
		return fmt.Errorf("create temp lock file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup; no-op after a successful rename

	if _, err := tmp.Write(data); err != nil {
		if closeErr := tmp.Close(); closeErr != nil {
			return fmt.Errorf("write temp lock file: %w", errors.Join(err, closeErr))
		}
		return fmt.Errorf("write temp lock file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp lock file: %w", err)
	}
	if err := chmodLockTempFile(tmpPath, perm); err != nil {
		return fmt.Errorf("chmod temp lock file: %w", err)
	}
	if err := os.Rename(tmpPath, lockPath); err != nil {
		return fmt.Errorf("rename lock into place: %w", err)
	}
	return nil
}

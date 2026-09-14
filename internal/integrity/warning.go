// Package integrity provides config drift-detection helpers for the Strategist CLI.
//
// This is operational drift detection, not tamper-proof security: a local actor
// who can edit both active.yaml and .config.lock can reseal malicious changes.
// The goal is to catch accidental or manual edits made outside the CLI.
package integrity

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// lockSchemaV1 identifies the current lock JSON shape. Locks written before this
// field existed ("legacy" locks) contain only mtime and path.
const lockSchemaV1 = "strategist-config-lock/1.0"

// Reason explains why a Result reports (or does not report) drift.
type Reason string

// Reason values describe config lock comparison outcomes. ReasonUnmodified is
// this package's "no drift detected" result — it covers only the byte
// (SHA256/size) and provenance (did this file change since the CLI itself
// sealed it) drift classes, not schema, contract, behavior, or semantic
// drift. See docs/drift-detection-matrix.md for the full per-detector
// breakdown, including this package's role in strategist check.
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

// statConfig, lockPathMismatch, pathMismatchResult, compareConfigState,
// configDriftReason, lockHashMismatch, lockMTimeMismatch, readLock, hashFile,
// and normalizePath (Check's internal drift-comparison helpers) live in
// warning_compare.go, split out to keep this file under the repo's
// file-size budget.
//
// Atomic lock-file writing (lockTempFile, createLockTempFile,
// chmodLockTempFile, writeLockFileAtomically) lives in
// warning_atomic_write.go, for the same reason.

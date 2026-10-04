package dojo

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	storageLockName    = ".state.lock"
	storageLockTimeout = 5 * time.Second
	storageLockStale   = 30 * time.Second
	storageLockRetry   = 10 * time.Millisecond
)

// withStorageLock serializes state writes across goroutines and independent
// strategist processes. The lock is a short-lived directory created atomically;
// an old lock is recoverable because all writes are designed to be bounded.
func withStorageLock(paths StoragePaths, fn func() error) error {
	lockPath, err := acquireStorageLock(paths)
	if err != nil {
		return err
	}
	return releaseStorageLock(lockPath, fn())
}

func acquireStorageLock(paths StoragePaths) (string, error) {
	if err := os.MkdirAll(paths.DojoRoot, 0o755); err != nil { //nolint:gosec // G301: dojo storage domain
		return "", fmt.Errorf("dojo: create %s: %w", paths.DojoRoot, err)
	}
	lockPath := filepath.Join(paths.DojoRoot, storageLockName)
	deadline := time.Now().Add(storageLockTimeout)
	for time.Now().Before(deadline) {
		if err := os.Mkdir(lockPath, 0o700); err == nil { //nolint:gosec // G301: transient dojo lock
			return lockPath, nil
		} else if !os.IsExist(err) {
			return "", fmt.Errorf("dojo: acquire storage lock: %w", err)
		}
		if recoverStaleStorageLock(lockPath) {
			continue
		}
		time.Sleep(storageLockRetry)
	}
	return "", fmt.Errorf("dojo: storage lock %s is busy", lockPath)
}

func recoverStaleStorageLock(lockPath string) bool {
	info, err := os.Stat(lockPath)
	if err != nil || time.Since(info.ModTime()) <= storageLockStale {
		return false
	}
	removeErr := os.Remove(lockPath)
	return removeErr == nil || os.IsNotExist(removeErr)
}

func releaseStorageLock(lockPath string, workErr error) error {
	removeErr := os.Remove(lockPath)
	if removeErr != nil && !os.IsNotExist(removeErr) {
		removeErr = fmt.Errorf("dojo: release storage lock: %w", removeErr)
	}
	return errors.Join(workErr, removeErr)
}

func atomicWriteFile(path string, data []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*") //nolint:gosec // G304: path is inside dojo storage
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }() //nolint:errcheck // best-effort cleanup after atomic write
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close() //nolint:errcheck // preserve the primary chmod error
		return fmt.Errorf("chmod temporary file: %w", err)
	}
	if n, err := tmp.Write(data); err != nil {
		_ = tmp.Close() //nolint:errcheck // preserve the primary write error
		return fmt.Errorf("write temporary file: %w", err)
	} else if n != len(data) {
		_ = tmp.Close() //nolint:errcheck // preserve the short-write error
		return io.ErrShortWrite
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close() //nolint:errcheck // preserve the primary sync error
		return fmt.Errorf("sync temporary file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temporary file: %w", err)
	}
	return nil
}

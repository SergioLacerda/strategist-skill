package mission

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// invocationClaimTTL bounds how long a crashed completion can keep a request
// locked before another caller may reclaim it.
const invocationClaimTTL = 10 * time.Minute

// Claim takes an exclusive, crash-recoverable lock on one request so two
// completions cannot normalize and persist the same request concurrently. The
// returned release function is safe to call more than once.
func (s InvocationStore) Claim(requestID string) (func(), error) {
	path, err := s.claimPath(requestID)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("mission invocation: create request directory: %w", err)
	}
	if err := s.createClaim(path); err != nil {
		return nil, err
	}
	return func() { _ = os.Remove(path) }, nil //nolint:errcheck // best-effort release; stale claims expire by TTL.
}

func (s InvocationStore) createClaim(path string) error {
	err := tryCreateClaim(path)
	if errors.Is(err, os.ErrExist) && s.reclaimStale(path) {
		err = tryCreateClaim(path)
	}
	if errors.Is(err, os.ErrExist) {
		return fmt.Errorf("invocation_in_progress: another completion holds this request")
	}
	return err
}

func tryCreateClaim(path string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) //nolint:gosec // G304: path derives from a validated request id.
	if err != nil {
		return fmt.Errorf("mission invocation: claim request: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("mission invocation: close claim: %w", err)
	}
	return nil
}

func (s InvocationStore) reclaimStale(path string) bool {
	info, err := os.Stat(path)
	if err != nil || s.now().Sub(info.ModTime()) <= invocationClaimTTL {
		return false
	}
	return os.Remove(path) == nil
}

func (s InvocationStore) claimPath(requestID string) (string, error) {
	path, err := s.path(requestID)
	if err != nil {
		return "", err
	}
	return path[:len(path)-len(".json")] + ".claim", nil
}

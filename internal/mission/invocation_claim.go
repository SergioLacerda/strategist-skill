package mission

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// invocationClaimTTL bounds how long a crashed completion can keep a target
// leased before another caller may reclaim it, once its owner is gone.
const invocationClaimTTL = 10 * time.Minute

// ClaimTarget takes an exclusive, crash-recoverable lease on one publication
// target (mission, Role and slot), so two requests for the same artifact can
// never publish concurrently, whatever their request IDs. The returned release
// function is safe to call more than once.
func (s InvocationStore) ClaimTarget(missionID, role, slot string) (func(), error) {
	path, err := s.targetLeasePath(missionID, role, slot)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("mission invocation: create lease directory: %w", err)
	}
	if err := s.createClaim(path); err != nil {
		return nil, err
	}
	return func() { _ = os.Remove(path) }, nil //nolint:errcheck // best-effort release; stale leases are reclaimed once their owner is gone.
}

func (s InvocationStore) createClaim(path string) error {
	err := tryCreateClaim(path)
	if errors.Is(err, os.ErrExist) && s.reclaimStale(path) {
		err = tryCreateClaim(path)
	}
	if errors.Is(err, os.ErrExist) {
		return fmt.Errorf("invocation_in_progress: another completion holds this mission target")
	}
	return err
}

func tryCreateClaim(path string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) //nolint:gosec // G304: path derives from a hash of the target identity.
	if err != nil {
		return fmt.Errorf("mission invocation: claim target: %w", err)
	}
	if _, err := file.WriteString(strconv.Itoa(os.Getpid())); err != nil {
		_ = file.Close() //nolint:errcheck // the write error is the one worth reporting.
		return fmt.Errorf("mission invocation: write lease owner: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("mission invocation: close lease: %w", err)
	}
	return nil
}

// reclaimStale removes a lease older than the TTL unless its recorded owner is
// still a live process; age alone never steals a lease from a live owner.
func (s InvocationStore) reclaimStale(path string) bool {
	info, err := os.Stat(path)
	if err != nil || s.now().Sub(info.ModTime()) <= invocationClaimTTL {
		return false
	}
	if raw, err := os.ReadFile(path); err == nil { //nolint:gosec // G304: path is a lease built by targetLeasePath.
		if pid, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil && processAlive(pid) {
			return false
		}
	}
	return os.Remove(path) == nil
}

func (s InvocationStore) targetLeasePath(missionID, role, slot string) (string, error) {
	if s.Root == "" {
		return "", fmt.Errorf("mission invocation: root is required")
	}
	if missionID == "" || role == "" || slot == "" {
		return "", fmt.Errorf("mission invocation: mission, role and slot are required for a target lease")
	}
	sum := sha256.Sum256([]byte(missionID + "\x00" + role + "\x00" + slot))
	return filepath.Join(s.Root, "missions", "invocations", "targets", hex.EncodeToString(sum[:16])+".lease"), nil
}

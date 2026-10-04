package mission

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/filelock"
)

// missionIDPattern protects filesystem lookups keyed by an operator-supplied
// mission id from path traversal and glob metacharacter injection.
var missionIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// ValidateMissionID validates the stable identifier used by mission artifacts.
func ValidateMissionID(id string) error {
	if id == "" {
		return errors.New("--mission-id is required")
	}
	if !missionIDPattern.MatchString(id) {
		return fmt.Errorf("--mission-id %q is malformed (want lowercase letters, digits, and hyphens, e.g. 20260830-skill-gaps-triage)", id)
	}
	return nil
}

// Path returns the canonical durable state path for one mission.
func Path(root, id string) string { return filepath.Join(root, "missions", id+".json") }

// RequireNoExisting rejects a mission id that already has durable state.
func RequireNoExisting(root, missionID string) error {
	if _, err := os.Stat(Path(root, missionID)); err == nil {
		return fmt.Errorf("mission %q already exists", missionID)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect existing state: %w", err)
	}
	return nil
}

// Save persists status atomically, so readers observe either the complete
// previous state or the complete new state, never a partially written JSON
// document.
func Save(root string, status domain.MissionEngineStatus) error {
	dir := filepath.Join(root, "missions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create mission directory: %w", err)
	}
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("encode mission state: %w", err)
	}
	target := Path(root, status.MissionID)
	tmp, err := os.CreateTemp(dir, filepath.Base(target)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create mission state temp file: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()        //nolint:errcheck // best-effort cleanup; the write error above is what's reported
		_ = os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup of an incomplete state file
		return fmt.Errorf("write mission state: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup of an incomplete state file
		return fmt.Errorf("close mission state temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		_ = os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup of an incomplete state file
		return fmt.Errorf("set mission state permissions: %w", err)
	}
	if err := os.Rename(tmpPath, target); err != nil {
		_ = os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup of an incomplete state file
		return fmt.Errorf("write mission state: %w", err)
	}
	return nil
}

// Lock serializes a mission read-modify-write critical section.
func Lock(root, missionID string, fn func() error) error {
	return filelock.WithLock(Path(root, missionID), fn) //nolint:wrapcheck // preserve the callback and filelock error categories
}

// Load reads and restores one durable mission state.
func Load(root, id string) (*domain.MissionEngine, domain.MissionEngineStatus, error) {
	data, err := os.ReadFile(Path(root, id))
	if err != nil {
		return nil, domain.MissionEngineStatus{}, fmt.Errorf("mission %q not found: %w", id, err)
	}
	var status domain.MissionEngineStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, status, fmt.Errorf("invalid persisted state: %w", err)
	}
	engine, err := domain.RestoreMission(status)
	if err != nil {
		return nil, status, fmt.Errorf("restore mission state: %w", err)
	}
	return engine, status, nil
}

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/filelock"
	"github.com/spf13/cobra"
)

// missionIDPattern restricts --mission-id to the same safe character set
// GenerateMissionID (internal/install/mission_id.go) produces:
// lowercase/digits/hyphens. This also protects filesystem lookups keyed by
// the id (missionPath, missionIDKnown's filepath.Glob) from path-traversal or
// glob-metacharacter injection via an operator-supplied string.
var missionIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// validateMissionID is the single --mission-id check shared by every mission
// subcommand. It returns an unprefixed error; callers add their command prefix.
func validateMissionID(id string) error {
	if id == "" {
		return errors.New("--mission-id is required")
	}
	if !missionIDPattern.MatchString(id) {
		return fmt.Errorf("--mission-id %q is malformed (want lowercase letters, digits, and hyphens, e.g. 20260830-skill-gaps-triage)", id)
	}
	return nil
}

func requireMissionID(id string) error {
	if err := validateMissionID(id); err != nil {
		return fmt.Errorf("mission: %w", err)
	}
	return nil
}

func missionPath(root, id string) string { return filepath.Join(root, "missions", id+".json") }

// saveMission persists status atomically: it writes to a sibling temp file
// and renames it over the target (ADR-0057 § D1). os.Rename is atomic within
// a filesystem on POSIX, so a concurrent reader (or a process interrupted
// mid-write) always sees either the complete previous state or the complete
// new state — never a truncated file that loadMission would reject as
// "invalid persisted state".
func saveMission(root string, status domain.MissionEngineStatus) error {
	dir := filepath.Join(root, "missions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create mission directory: %w", err)
	}
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("encode mission state: %w", err)
	}
	target := missionPath(root, status.MissionID)
	tmp, err := os.CreateTemp(dir, filepath.Base(target)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create mission state temp file: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()        //nolint:errcheck // best-effort cleanup; the write error above is what's reported
		_ = os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup; a leaked .tmp-* file is harmless and self-evident
		return fmt.Errorf("write mission state: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup; the close error above is what's reported
		return fmt.Errorf("close mission state temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		_ = os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup; the chmod error above is what's reported
		return fmt.Errorf("set mission state permissions: %w", err)
	}
	if err := os.Rename(tmpPath, target); err != nil {
		_ = os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup; the rename error above is what's reported
		return fmt.Errorf("write mission state: %w", err)
	}
	return nil
}

// lockMission serializes the mission's read-modify-write critical section
// (RequireNoExisting/Load through Save) against concurrent CLI invocations
// for the same mission id (ADR-0057 § D2). It generalizes
// internal/leveling's own ledger lock into internal/filelock rather than
// introducing a second flock implementation.
func lockMission(root, missionID string, fn func() error) error {
	return filelock.WithLock(missionPath(root, missionID), fn) //nolint:wrapcheck // the result is fn()'s own already-meaningful error on the common path; filelock's own lock-acquisition errors are already prefixed "filelock: " internally
}

func loadMission(root, id string) (*domain.MissionEngine, domain.MissionEngineStatus, error) {
	data, err := os.ReadFile(missionPath(root, id))
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

func writeMissionResult(cmd *cobra.Command, asJSON bool, value any) error {
	if asJSON {
		return encodeMissionResult(cmd, value)
	}
	status, ok := value.(domain.MissionEngineStatus)
	if !ok {
		return encodeMissionResult(cmd, value)
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "mission_id=%s phase=%s state=%s\n", status.MissionID, status.Phase, status.State); err != nil {
		return fmt.Errorf("write mission result: %w", err)
	}
	return nil
}

func encodeMissionResult(cmd *cobra.Command, value any) error {
	if err := json.NewEncoder(cmd.OutOrStdout()).Encode(value); err != nil {
		return fmt.Errorf("encode mission result: %w", err)
	}
	return nil
}

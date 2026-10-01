package handoff

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// OutcomeStore persists Outcome records below one Strategist runtime root:
// missions/handoff/<mission_id>/attempt-<n>.json, plus a consumed marker once
// execution entry used the outcome. Each attempt file is created exclusively,
// so an attempt can never be silently replaced.
type OutcomeStore struct {
	Root  string
	Clock func() time.Time
}

// NewOutcomeStore creates a store with a UTC clock.
func NewOutcomeStore(root string) OutcomeStore {
	return OutcomeStore{Root: root, Clock: func() time.Time { return time.Now().UTC() }}
}

func (s OutcomeStore) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock().UTC()
}

func (s OutcomeStore) dir(missionID string) (string, error) {
	if s.Root == "" {
		return "", fmt.Errorf("handoff_outcome_invalid: runtime root is required")
	}
	if missionID == "" || strings.ContainsAny(missionID, `/\`) || missionID == "." || missionID == ".." {
		return "", fmt.Errorf("handoff_outcome_invalid: malformed mission id %q", missionID)
	}
	return filepath.Join(s.Root, "missions", "handoff", missionID), nil
}

func attemptFile(attempt int) string { return fmt.Sprintf("attempt-%03d.json", attempt) }

// NextAttempt is the attempt number the next outcome must carry.
func (s OutcomeStore) NextAttempt(missionID string) (int, error) {
	attempts, err := s.attempts(missionID)
	if err != nil {
		return 0, err
	}
	if len(attempts) == 0 {
		return 1, nil
	}
	return attempts[len(attempts)-1] + 1, nil
}

func (s OutcomeStore) attempts(missionID string) ([]int, error) {
	dir, err := s.dir(missionID)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("handoff_outcome_unreadable: list outcomes: %w", err)
	}
	return attemptNumbers(entries)
}

// attemptNumbers returns the ascending attempt numbers of attempt-<n>.json
// entries; any other attempt-* name is tampering. Temporary and marker files
// are ignored.
func attemptNumbers(entries []os.DirEntry) ([]int, error) {
	var attempts []int
	for _, entry := range entries {
		number, isAttempt, err := parseAttemptName(entry.Name())
		if err != nil {
			return nil, err
		}
		if isAttempt {
			attempts = append(attempts, number)
		}
	}
	sort.Ints(attempts)
	return attempts, nil
}

func parseAttemptName(name string) (int, bool, error) {
	rest, ok := strings.CutPrefix(name, "attempt-")
	if !ok {
		return 0, false, nil
	}
	number, err := strconv.Atoi(strings.TrimSuffix(rest, ".json"))
	if err != nil || number < 1 {
		return 0, false, fmt.Errorf("handoff_outcome_tampered: unrecognized outcome file %q", name)
	}
	return number, true, nil
}

// Append seals and durably writes the next attempt's outcome. A duplicate or
// out-of-sequence attempt, or any write failure, is an error and leaves no
// partial record behind.
func (s OutcomeStore) Append(outcome Outcome) (Outcome, error) {
	sealed, err := outcome.Seal(s.now())
	if err != nil {
		return Outcome{}, err
	}
	next, err := s.NextAttempt(sealed.MissionID)
	if err != nil {
		return Outcome{}, err
	}
	if sealed.Attempt != next {
		return Outcome{}, fmt.Errorf("handoff_outcome_attempt_conflict: attempt %d is not the next attempt %d for mission %q", sealed.Attempt, next, sealed.MissionID)
	}
	raw, err := json.MarshalIndent(sealed, "", "  ")
	if err != nil {
		return Outcome{}, fmt.Errorf("handoff_outcome_persist_failed: encode outcome: %w", err)
	}
	dir, err := s.dir(sealed.MissionID)
	if err != nil {
		return Outcome{}, err
	}
	if err := linkExclusive(dir, attemptFile(sealed.Attempt), append(raw, '\n')); err != nil {
		return Outcome{}, fmt.Errorf("handoff_outcome_persist_failed: %w", err)
	}
	return sealed, nil
}

// linkExclusive writes content to a temporary file and links it to its final
// name, which fails if the name exists, so a record appears whole or not at all.
func linkExclusive(dir, name string, content []byte) error {
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // G301: runtime state directory
		return fmt.Errorf("create outcome directory: %w", err)
	}
	temp, err := writeTemporary(dir, name, content)
	if err != nil {
		return err
	}
	linkErr := os.Link(temp, filepath.Join(dir, name))
	if removeErr := os.Remove(temp); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) && linkErr == nil {
		return fmt.Errorf("remove temporary outcome: %w", removeErr)
	}
	if linkErr != nil {
		return fmt.Errorf("publish outcome %s: %w", name, linkErr)
	}
	return nil
}

func writeTemporary(dir, name string, content []byte) (string, error) {
	temp, err := os.CreateTemp(dir, ".tmp-"+name+"-")
	if err != nil {
		return "", fmt.Errorf("create temporary outcome: %w", err)
	}
	if _, err := temp.Write(content); err != nil {
		_ = temp.Close()           //nolint:errcheck // the write error is the one worth reporting
		_ = os.Remove(temp.Name()) //nolint:errcheck // best-effort cleanup of the failed temporary
		return "", fmt.Errorf("write outcome: %w", err)
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(temp.Name()) //nolint:errcheck // best-effort cleanup of the failed temporary
		return "", fmt.Errorf("close outcome: %w", err)
	}
	return temp.Name(), nil
}

// Latest loads the highest-numbered attempt and verifies its integrity. It
// reports handoff_outcome_missing when no attempt exists.
func (s OutcomeStore) Latest(missionID string) (Outcome, error) {
	attempts, err := s.attempts(missionID)
	if err != nil {
		return Outcome{}, err
	}
	if len(attempts) == 0 {
		return Outcome{}, fmt.Errorf("handoff_outcome_missing: no Archivist-to-Sniper outcome is recorded for mission %q; run `strategist handoff verify --transition archivist_to_sniper` first", missionID)
	}
	dir, err := s.dir(missionID)
	if err != nil {
		return Outcome{}, err
	}
	raw, err := os.ReadFile(filepath.Join(dir, attemptFile(attempts[len(attempts)-1]))) //nolint:gosec // G304: path is derived from a validated mission id
	if err != nil {
		return Outcome{}, fmt.Errorf("handoff_outcome_unreadable: %w", err)
	}
	var outcome Outcome
	if err := json.Unmarshal(raw, &outcome); err != nil {
		return Outcome{}, fmt.Errorf("handoff_outcome_tampered: outcome is not valid JSON: %w", err)
	}
	if err := outcome.VerifyIntegrity(); err != nil {
		return Outcome{}, err
	}
	return outcome, nil
}

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

// OutcomeStore persists Outcome records below one Strategist runtime root. The
// established Archivist-to-Sniper path keeps its original location at
// missions/handoff/<mission_id>/attempt-<n>.json; Ranger-to-Archivist uses a
// transition-specific child directory. Each attempt file is exclusive.
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
	return s.dirFor(missionID, TransitionArchivistToSniper)
}

func (s OutcomeStore) dirFor(missionID, transition string) (string, error) {
	if s.Root == "" {
		return "", fmt.Errorf("handoff_outcome_invalid: runtime root is required")
	}
	if missionID == "" || strings.ContainsAny(missionID, `/\`) || missionID == "." || missionID == ".." {
		return "", fmt.Errorf("handoff_outcome_invalid: malformed mission id %q", missionID)
	}
	if transition == "" || transition == TransitionArchivistToSniper {
		return filepath.Join(s.Root, "missions", "handoff", missionID), nil
	}
	if transition != TransitionRangerToArchivist {
		return "", fmt.Errorf("handoff_outcome_invalid: transition %q is not lifecycle-owned", transition)
	}
	return filepath.Join(s.Root, "missions", "handoff", missionID, transition), nil
}

func attemptFile(attempt int) string { return fmt.Sprintf("attempt-%03d.json", attempt) }

// NextAttempt is the attempt number the next outcome must carry.
func (s OutcomeStore) NextAttempt(missionID string) (int, error) {
	return s.NextAttemptFor(missionID, TransitionArchivistToSniper)
}

// NextAttemptFor returns the next attempt for one lifecycle transition.
func (s OutcomeStore) NextAttemptFor(missionID, transition string) (int, error) {
	attempts, err := s.attemptsFor(missionID, transition)
	if err != nil {
		return 0, err
	}
	if len(attempts) == 0 {
		return 1, nil
	}
	return attempts[len(attempts)-1] + 1, nil
}

func (s OutcomeStore) attemptsFor(missionID, transition string) ([]int, error) {
	dir, err := s.dirFor(missionID, transition)
	if err != nil {
		return nil, err
	}
	info, statErr := os.Stat(dir)
	if errors.Is(statErr, os.ErrNotExist) {
		return nil, nil
	}
	if statErr != nil {
		return nil, fmt.Errorf("handoff_outcome_unreadable: list outcomes: %w", statErr)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("handoff_outcome_unreadable: list outcomes: %s is not a directory", dir)
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
	next, err := s.NextAttemptFor(sealed.MissionID, sealed.Transition)
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
	dir, err := s.dirFor(sealed.MissionID, sealed.Transition)
	if err != nil {
		return Outcome{}, err
	}
	if err := linkExclusive(dir, attemptFile(sealed.Attempt), append(raw, '\n')); err != nil {
		return Outcome{}, fmt.Errorf("handoff_outcome_persist_failed: %w", err)
	}
	return sealed, nil
}

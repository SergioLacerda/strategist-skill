package handoff

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Latest loads the highest-numbered attempt and verifies its integrity. It
// reports handoff_outcome_missing when no attempt exists.
func (s OutcomeStore) Latest(missionID string) (Outcome, error) {
	return s.LatestFor(missionID, TransitionArchivistToSniper)
}

// Outcomes returns the immutable, integrity-checked audit history for one
// lifecycle transition. Callers use the global sequence for audit ordering but
// must not mistake it for a retry budget of a later package revision.
func (s OutcomeStore) Outcomes(missionID string) ([]Outcome, error) {
	return s.OutcomesFor(missionID, TransitionArchivistToSniper)
}

// OutcomesFor returns the integrity-checked audit history for transition.
func (s OutcomeStore) OutcomesFor(missionID, transition string) ([]Outcome, error) {
	attempts, err := s.attemptsFor(missionID, transition)
	if err != nil {
		return nil, err
	}
	dir, err := s.dirFor(missionID, transition)
	if err != nil {
		return nil, err
	}
	outcomes := make([]Outcome, 0, len(attempts))
	for _, attempt := range attempts {
		outcome, readErr := readVerifiedOutcome(filepath.Join(dir, attemptFile(attempt)))
		if readErr != nil {
			return nil, readErr
		}
		outcomes = append(outcomes, outcome)
	}
	return outcomes, nil
}

// LatestFor loads and verifies the latest outcome for one transition.
func (s OutcomeStore) LatestFor(missionID, transition string) (Outcome, error) {
	attempts, err := s.attemptsFor(missionID, transition)
	if err != nil {
		return Outcome{}, err
	}
	if len(attempts) == 0 {
		return Outcome{}, missingOutcomeError(missionID, transition)
	}
	dir, err := s.dirFor(missionID, transition)
	if err != nil {
		return Outcome{}, err
	}
	return readVerifiedOutcome(filepath.Join(dir, attemptFile(attempts[len(attempts)-1])))
}

func missingOutcomeError(missionID, transition string) error {
	if transition == TransitionRangerToArchivist {
		return fmt.Errorf("handoff_outcome_missing: no Ranger-to-Archivist outcome is recorded for mission %q; run `strategist handoff evaluate-ranger` first", missionID)
	}
	return fmt.Errorf("handoff_outcome_missing: no Archivist-to-Sniper outcome is recorded for mission %q; run `strategist handoff verify --transition archivist_to_sniper` first", missionID)
}

// readVerifiedOutcome loads one attempt file and checks its integrity digest.
func readVerifiedOutcome(path string) (Outcome, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // G304: path is derived from a validated mission id
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

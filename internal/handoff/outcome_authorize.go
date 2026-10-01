package handoff

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type consumedMarker struct {
	Integrity  string `json:"outcome_integrity"`
	ConsumedAt string `json:"consumed_at"`
}

const consumedFile = "consumed.json"

// Consumed reports whether execution entry already used this outcome.
func (s OutcomeStore) Consumed(outcome Outcome) (bool, error) {
	dir, err := s.dir(outcome.MissionID)
	if err != nil {
		return false, err
	}
	raw, err := os.ReadFile(filepath.Join(dir, consumedFile)) //nolint:gosec // G304: path is derived from a validated mission id
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("handoff_outcome_unreadable: %w", err)
	}
	var marker consumedMarker
	if err := json.Unmarshal(raw, &marker); err != nil {
		return false, fmt.Errorf("handoff_outcome_tampered: consumed marker is not valid JSON: %w", err)
	}
	return marker.Integrity == outcome.Integrity, nil
}

// Consume marks the outcome used. It is exclusive, so a second consumption of
// the same mission fails.
func (s OutcomeStore) Consume(outcome Outcome) error {
	dir, err := s.dir(outcome.MissionID)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(consumedMarker{Integrity: outcome.Integrity, ConsumedAt: s.now().Format(time.RFC3339Nano)})
	if err != nil {
		return fmt.Errorf("handoff_outcome_persist_failed: encode consumed marker: %w", err)
	}
	if err := linkExclusive(dir, consumedFile, append(raw, '\n')); err != nil {
		return fmt.Errorf("handoff_outcome_persist_failed: %w", err)
	}
	return nil
}

// ExecutionCheck is what the execution boundary knows about the current
// package; the outcome must correlate with all of it.
type ExecutionCheck struct {
	MissionID      string
	PackageDigest  string
	PolicyID       string
	SkipAuthorized bool
}

// AuthorizeExecution loads the current outcome and returns it only when it is
// intact, belongs to this mission and transition, was decided for the current
// package under the current policy, has not been used, and is either passed or
// a skip the current signals still authorize. Anything else denies.
func (s OutcomeStore) AuthorizeExecution(check ExecutionCheck) (Outcome, error) {
	outcome, err := s.Latest(check.MissionID)
	if err != nil {
		return Outcome{}, err
	}
	if err := correlateOutcome(outcome, check); err != nil {
		return Outcome{}, err
	}
	consumed, err := s.Consumed(outcome)
	if err != nil {
		return Outcome{}, err
	}
	if consumed {
		return Outcome{}, fmt.Errorf("handoff_outcome_replayed: the outcome for mission %q attempt %d was already used to enter execution", outcome.MissionID, outcome.Attempt)
	}
	return outcome, nil
}

func correlateOutcome(outcome Outcome, check ExecutionCheck) error {
	switch {
	case outcome.MissionID != check.MissionID:
		return fmt.Errorf("handoff_outcome_cross_mission: outcome belongs to mission %q, not %q", outcome.MissionID, check.MissionID)
	case outcome.Transition != TransitionArchivistToSniper:
		return fmt.Errorf("handoff_outcome_transition_mismatch: outcome is for %q, not %q", outcome.Transition, TransitionArchivistToSniper)
	case outcome.PackageDigest != check.PackageDigest:
		return fmt.Errorf("handoff_outcome_stale: the refined package changed after the outcome was recorded (outcome %s, package %s); verify the handoff again", outcome.PackageDigest, check.PackageDigest)
	case outcome.PolicyID != check.PolicyID:
		return fmt.Errorf("handoff_outcome_policy_mismatch: the outcome was decided under a different handoff policy")
	}
	switch outcome.Result {
	case OutcomePassed:
		return nil
	case OutcomeSkipped:
		if !check.SkipAuthorized {
			return fmt.Errorf("handoff_outcome_skip_not_authorized: the current package facts no longer authorize skipping the challenge")
		}
		return nil
	case OutcomeFailed:
		return fmt.Errorf("handoff_outcome_failed: the recorded handoff challenge failed (attempt %d); repair the package and verify again", outcome.Attempt)
	default:
		return fmt.Errorf("handoff_outcome_unknown_result: %q does not authorize execution", outcome.Result)
	}
}

package handoff

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const invalidatedFilePrefix = "invalidated-"

type invalidatedMarker struct {
	OutcomeIntegrity string `json:"outcome_integrity"`
	PackageDigest    string `json:"package_digest,omitempty"`
	Reason           string `json:"reason"`
	InvalidatedAt    string `json:"invalidated_at"`
}

// InvalidateLatest marks the latest Archivist-to-Sniper outcome as unusable
// while retaining the original outcome for audit reconstruction. A mission
// without a recorded outcome has nothing to invalidate.
func (s OutcomeStore) InvalidateLatest(missionID, reason string) (bool, error) {
	attempts, err := s.attemptsFor(missionID, TransitionArchivistToSniper)
	if err != nil {
		return false, err
	}
	if len(attempts) == 0 {
		return false, nil
	}
	outcome, err := s.Latest(missionID)
	if err != nil {
		return false, err
	}
	if err := s.Invalidate(outcome, reason); err != nil {
		return false, err
	}
	return true, nil
}

// Invalidate writes an exclusive marker for one outcome. The outcome remains
// immutable and can still be inspected, but AuthorizeExecution will reject it
// until a new gate and handoff produce a later outcome.
func (s OutcomeStore) Invalidate(outcome Outcome, reason string) error {
	if err := outcome.VerifyIntegrity(); err != nil {
		return err
	}
	if reason == "" {
		return fmt.Errorf("handoff_outcome_invalid: invalidation reason is required")
	}
	marker := invalidatedMarker{
		OutcomeIntegrity: outcome.Integrity,
		PackageDigest:    outcome.PackageDigest,
		Reason:           reason,
		InvalidatedAt:    s.now().Format("2006-01-02T15:04:05.999999999Z07:00"),
	}
	raw, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return fmt.Errorf("handoff_outcome_persist_failed: encode invalidation: %w", err)
	}
	dir, err := s.dirFor(outcome.MissionID, outcome.Transition)
	if err != nil {
		return err
	}
	return linkExclusive(dir, invalidatedFile(outcome.Attempt), append(raw, '\n'))
}

func invalidatedFile(attempt int) string {
	return fmt.Sprintf("%s%03d.json", invalidatedFilePrefix, attempt)
}

func (s OutcomeStore) isInvalidated(outcome Outcome) (bool, error) {
	dir, err := s.dirFor(outcome.MissionID, outcome.Transition)
	if err != nil {
		return false, err
	}
	raw, err := os.ReadFile(filepath.Join(dir, invalidatedFile(outcome.Attempt))) //nolint:gosec // G304: path is derived from validated outcome identity
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("handoff_outcome_unreadable: read invalidation: %w", err)
	}
	var marker invalidatedMarker
	if err := json.Unmarshal(raw, &marker); err != nil {
		return false, fmt.Errorf("handoff_outcome_tampered: invalidation marker is not valid JSON: %w", err)
	}
	if marker.OutcomeIntegrity != outcome.Integrity || marker.PackageDigest != outcome.PackageDigest || marker.Reason == "" || marker.InvalidatedAt == "" {
		return false, fmt.Errorf("handoff_outcome_tampered: invalidation marker does not match mission %q attempt %d", outcome.MissionID, outcome.Attempt)
	}
	return true, nil
}

// Invalidated reports whether an outcome has an audit-preserving invalidation
// marker and therefore cannot authorize execution.
func (s OutcomeStore) Invalidated(outcome Outcome) (bool, error) {
	return s.isInvalidated(outcome)
}

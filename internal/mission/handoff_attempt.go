package mission

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
)

// nextHandoffAttempt numbers a new evaluation after a failed or explicitly
// invalidated outcome, while refusing replay of a live passed/skipped outcome.
func nextHandoffAttempt(store handoff.OutcomeStore, missionID, digest string, policy handoff.Policy) (int, error) {
	attempt, err := store.NextAttempt(missionID)
	if err != nil {
		return 0, fmt.Errorf("evaluate handoff: %w", err)
	}
	if attempt == 1 {
		return 1, nil
	}
	latest, err := store.Latest(missionID)
	if err != nil {
		return 0, fmt.Errorf("evaluate handoff: %w", err)
	}
	if latest.Result != handoff.OutcomeFailed {
		return nextAfterNonFailedOutcome(store, latest, attempt, missionID, digest, policy)
	}
	return checkedHandoffAttempt(attempt, missionID, policy)
}

func nextAfterNonFailedOutcome(store handoff.OutcomeStore, latest handoff.Outcome, attempt int, missionID, digest string, policy handoff.Policy) (int, error) {
	invalidated, err := store.Invalidated(latest)
	if err != nil {
		return 0, fmt.Errorf("evaluate handoff: %w", err)
	}
	if invalidated {
		return checkedHandoffAttempt(attempt, missionID, policy)
	}
	return 0, unfailedOutcomeError(latest, digest)
}

func checkedHandoffAttempt(attempt int, missionID string, policy handoff.Policy) (int, error) {
	if policy.MaxAttempts > 0 && attempt > policy.MaxAttempts {
		return 0, fmt.Errorf("handoff_attempts_exhausted: mission %q used all %d attempts", missionID, policy.MaxAttempts)
	}
	return attempt, nil
}

func unfailedOutcomeError(latest handoff.Outcome, digest string) error {
	if latest.PackageDigest != digest {
		return fmt.Errorf("handoff_package_changed_after_outcome: mission %q has a %s outcome (attempt %d) for another package revision; return to refinement with handoff_challenge_failed and accept the Approval Gate again", latest.MissionID, latest.Result, latest.Attempt)
	}
	return fmt.Errorf("handoff_outcome_already_recorded: mission %q already has a %s outcome for this package revision (attempt %d)", latest.MissionID, latest.Result, latest.Attempt)
}

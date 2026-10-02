package mission

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
)

// AuthorizeHandoffExecution is the execution-entry check. It re-derives the
// current package identity, signals and policy and loads the mission's latest
// outcome, returning it only when it authorizes this package right now: intact,
// same mission and transition, same package revision and policy, not already
// used, and either passed or a skip the current facts still authorize. It does
// not look at the Approval Gate, which stays an independent prerequisite.
func AuthorizeHandoffExecution(strategistRoot, basePath string, status domain.MissionEngineStatus) (handoff.Outcome, error) {
	refined := refinedDir(basePath, status.MissionID)
	extracted, err := handoff.ExtractRiskSignals(refined, status.MissionID, "")
	if err != nil {
		return handoff.Outcome{}, fmt.Errorf("authorize execution: %w", err)
	}
	digest, err := handoff.PackageDigest(refined)
	if err != nil {
		return handoff.Outcome{}, fmt.Errorf("authorize execution: %w", err)
	}
	if err := requireApprovedPackageDigest(status, digest); err != nil {
		return handoff.Outcome{}, fmt.Errorf("authorize execution: %w", err)
	}
	policy := handoff.ResolveArchivistPolicy(extracted.Signals)
	if err := requireConsistentMetadata(refined, policy.Enabled); err != nil {
		return handoff.Outcome{}, fmt.Errorf("authorize execution: %w", err)
	}
	outcome, err := handoff.NewOutcomeStore(strategistRoot).AuthorizeExecution(handoff.ExecutionCheck{
		MissionID: status.MissionID, PackageDigest: digest, PolicyID: handoff.PolicyIdentity(policy), SkipAuthorized: !policy.Enabled,
	})
	if err != nil {
		return handoff.Outcome{}, fmt.Errorf("authorize execution: %w", err)
	}
	if err := requireConsistentMetadata(refined, outcome.Required); err != nil {
		return handoff.Outcome{}, fmt.Errorf("authorize execution: %w", err)
	}
	if outcome.Attempt != status.HandoffAttempt {
		return handoff.Outcome{}, fmt.Errorf("authorize execution: handoff_outcome_state_mismatch: the outcome is attempt %d but mission state records attempt %d; run `strategist handoff evaluate` to reconcile", outcome.Attempt, status.HandoffAttempt)
	}
	return outcome, nil
}

// requireConsistentMetadata fails closed when the package's optional
// handoff_verification metadata contradicts the package-derived policy. The
// metadata is a mirror, never proof, and its absence is fine.
func requireConsistentMetadata(refined string, required bool) error {
	metadata, err := handoff.ReadVerificationMetadata(refined)
	if err != nil {
		return fmt.Errorf("read handoff metadata: %w", err)
	}
	if err := metadata.ConsistentWith(required); err != nil {
		return fmt.Errorf("check handoff metadata: %w", err)
	}
	return nil
}

// ConsumeHandoffOutcome marks the outcome used, once execution was entered.
func ConsumeHandoffOutcome(strategistRoot string, outcome handoff.Outcome) error {
	if err := handoff.NewOutcomeStore(strategistRoot).Consume(outcome); err != nil {
		return fmt.Errorf("consume handoff outcome: %w", err)
	}
	return nil
}

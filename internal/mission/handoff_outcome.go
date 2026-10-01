package mission

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// ArchivistHandoffInput carries what the caller may supply for an evaluation:
// the coarse intake risk label (it can only make the result stricter) and, when
// the policy requires the challenge, the challenges and acknowledgment. Nothing
// here can declare a challenge unnecessary or pick the policy.
type ArchivistHandoffInput struct {
	RiskLevel  string
	Challenges []handoff.Challenge
	Ack        handoff.Acknowledgment
	// ConfidenceSummary is the optional declared handoff confidence. When nil,
	// an explicit missing-record is written instead; either way the record is
	// persisted before the outcome, so a recording failure authorizes nothing.
	ConfidenceSummary *domain.ConfidenceSummary
}

// ArchivistHandoffResult is the production evaluation of the
// Archivist-to-Sniper handoff and the durable outcome it recorded.
type ArchivistHandoffResult struct {
	Outcome handoff.Outcome
	Result  handoff.Result
	Policy  handoff.Policy
}

func refinedDir(basePath, missionID string) string {
	return filepath.Join(basePath, "refined", missionID)
}

// EvaluateArchivistHandoff is the one production boundary for the
// Archivist-to-Sniper handoff: it validates the refined package, derives the
// policy signals from its typed facts, resolves the policy, evaluates the
// required challenge (or authorizes a skip), and durably records a terminal
// outcome correlated to the mission, transition, package revision and attempt.
// It never advances the mission: entering execution is a separate submit that
// verifies and consumes this outcome. If the outcome cannot be persisted it
// returns handoff_outcome_persist_failed and nothing authorizes execution.
func EvaluateArchivistHandoff(strategistRoot, basePath string, status domain.MissionEngineStatus, input ArchivistHandoffInput) (ArchivistHandoffResult, error) {
	if status.State != domain.StateHandoffChallenge {
		return ArchivistHandoffResult{}, fmt.Errorf("handoff_gate_not_observed: mission %q is in state %q; the Approval Gate must be accepted before the handoff is evaluated", status.MissionID, status.State)
	}
	derived, err := deriveHandoff(basePath, status.MissionID, input.RiskLevel)
	if err != nil {
		return ArchivistHandoffResult{}, fmt.Errorf("evaluate handoff: %w", err)
	}
	store := handoff.NewOutcomeStore(strategistRoot)
	attempt, err := nextHandoffAttempt(store, status.MissionID, derived.digest, derived.policy)
	if err != nil {
		return ArchivistHandoffResult{}, err
	}
	result, err := challengeResult(strategistRoot, status, derived.policy, attempt, input)
	if err != nil {
		return ArchivistHandoffResult{}, err
	}
	if err := persistHandoffConfidence(strategistRoot, status.MissionID, attempt, input.ConfidenceSummary); err != nil {
		return ArchivistHandoffResult{}, fmt.Errorf("evaluate handoff: handoff_outcome_persist_failed: %w", err)
	}
	outcome, err := store.Append(derived.outcome(status, attempt, result))
	if err != nil {
		return ArchivistHandoffResult{}, fmt.Errorf("evaluate handoff: %w", err)
	}
	return ArchivistHandoffResult{Outcome: outcome, Result: result, Policy: derived.policy}, nil
}

// derivedHandoff is everything the package itself determines: its signals,
// content identity and the policy they resolve to.
type derivedHandoff struct {
	extracted handoff.ExtractedSignals
	digest    string
	policy    handoff.Policy
}

// deriveHandoff validates the refined package and derives its signals, content
// digest and policy, and rejects optional metadata that contradicts the policy.
func deriveHandoff(basePath, missionID, riskLevel string) (derivedHandoff, error) {
	refined := refinedDir(basePath, missionID)
	extracted, err := handoff.ExtractRiskSignals(refined, missionID, riskLevel)
	if err != nil {
		return derivedHandoff{}, fmt.Errorf("extract signals: %w", err)
	}
	digest, err := handoff.PackageDigest(refined)
	if err != nil {
		return derivedHandoff{}, fmt.Errorf("package digest: %w", err)
	}
	policy := handoff.ResolveArchivistPolicy(extracted.Signals)
	if err := requireConsistentMetadata(refined, policy.Enabled); err != nil {
		return derivedHandoff{}, err
	}
	return derivedHandoff{extracted: extracted, digest: digest, policy: policy}, nil
}

func (d derivedHandoff) outcome(status domain.MissionEngineStatus, attempt int, result handoff.Result) handoff.Outcome {
	return handoff.Outcome{
		MissionID: status.MissionID, Transition: handoff.TransitionArchivistToSniper, PackageDigest: d.digest, Attempt: attempt,
		Result: outcomeResult(d.policy, result), PolicyID: handoff.PolicyIdentity(d.policy), Required: d.policy.Enabled,
		Signals: handoff.SignalsRecordOf(d.extracted.Signals), Provenance: d.extracted.Provenance, GateObserved: string(status.State),
		ChallengeStatus: result.Status, CriticalFailures: result.CriticalFailures,
	}
}

// nextHandoffAttempt numbers the next evaluation. A new evaluation is only
// possible after a failed outcome (which returns the mission to refinement and
// a new Approval Gate) and while attempts remain: an unused passed or skipped
// outcome stands for the revision the gate approved, and a package changed
// after it must go back through refinement, never be re-evaluated in place.
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
		return 0, unfailedOutcomeError(latest, digest)
	}
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

// challengeResult runs the semantic challenge when the policy requires it, and
// records it in the challenge history. A skipped policy needs no challenge.
func challengeResult(strategistRoot string, status domain.MissionEngineStatus, policy handoff.Policy, attempt int, input ArchivistHandoffInput) (handoff.Result, error) {
	if !policy.Enabled {
		return handoff.Result{Status: handoff.StatusSkipped, Passed: true}, nil
	}
	if len(input.Challenges) == 0 {
		return handoff.Result{}, fmt.Errorf("handoff_challenge_required: the package requires the Archivist-to-Sniper challenge; supply the challenges and acknowledgment")
	}
	result := handoff.Verify(policy, input.Challenges, input.Ack)
	if result.Status == handoff.StatusPolicyInvalid {
		return handoff.Result{}, fmt.Errorf("handoff_policy_invalid: %v", result.PolicyErrors)
	}
	record := telemetry.ChallengeRecord{
		MissionID: status.MissionID, Transition: policy.Transition, Attempt: attempt, Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Status: result.Status, Passed: result.Passed, MissingRefs: result.MissingRefs, MissingChallenges: result.MissingChallenges,
		MisclassifiedRefs: result.MisclassifiedRefs, GateMismatch: result.GateMismatch, CounterfactualMismatches: result.CounterfactualMismatches,
		ForbiddenClaimViolations: result.ForbiddenClaimViolations, CriticalFailures: result.CriticalFailures,
	}
	if err := telemetry.AppendHandoffChallenge(telemetry.HandoffChallengeHistoryPath(strategistRoot), record); err != nil {
		return handoff.Result{}, fmt.Errorf("handoff_outcome_persist_failed: append challenge history: %w", err)
	}
	return result, nil
}

func outcomeResult(policy handoff.Policy, result handoff.Result) string {
	switch {
	case !policy.Enabled:
		return handoff.OutcomeSkipped
	case result.Passed:
		return handoff.OutcomePassed
	default:
		return handoff.OutcomeFailed
	}
}

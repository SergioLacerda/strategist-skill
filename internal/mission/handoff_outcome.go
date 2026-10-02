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
	if err := requireApprovedPackageDigest(status, derived.digest); err != nil {
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

// requireApprovedPackageDigest closes the pre-evaluation re-approval gap: the
// handoff may only evaluate the exact refined package whose digest was captured
// when the main Approval Gate was accepted.
func requireApprovedPackageDigest(status domain.MissionEngineStatus, current string) error {
	approved := status.ApprovalGatePackageDigest
	if approved == "" {
		return fmt.Errorf("handoff_gate_package_digest_missing: mission %q has no package digest captured at Approval Gate acceptance; accept the gate again", status.MissionID)
	}
	if approved != current {
		return fmt.Errorf("handoff_package_changed_after_gate: mission %q was approved for package %s but the current package is %s; accept the Approval Gate again", status.MissionID, approved, current)
	}
	return nil
}

func (d derivedHandoff) outcome(status domain.MissionEngineStatus, attempt int, result handoff.Result) handoff.Outcome {
	return handoff.Outcome{
		MissionID: status.MissionID, Transition: handoff.TransitionArchivistToSniper, PackageDigest: d.digest, Attempt: attempt,
		Result: outcomeResult(d.policy, result), PolicyID: handoff.PolicyIdentity(d.policy), Required: d.policy.Enabled,
		Signals: handoff.SignalsRecordOf(d.extracted.Signals), Provenance: d.extracted.Provenance, GateObserved: string(status.State),
		ChallengeStatus: result.Status, CriticalFailures: result.CriticalFailures,
	}
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

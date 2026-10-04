package mission

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// RangerHandoffInput is the lifecycle-owned challenge response. It is kept
// separate from the diagnostic handoff verify command so a diagnostic result
// can never become orchestration evidence by accident.
type RangerHandoffInput struct {
	Challenges []handoff.Challenge
	Ack        handoff.Acknowledgment
}

// RangerHandoffResult contains the persisted result and the deterministic
// policy evaluation used to produce it.
type RangerHandoffResult struct {
	Outcome handoff.Outcome
	Result  handoff.Result
	Policy  handoff.Policy
}

type rangerEvaluationContext struct {
	facts   handoff.RangerPolicyFacts
	policy  handoff.Policy
	digest  string
	attempt int
}

// RangerOutcomeAwaitingChallenge is the telemetry outcome emitted when Ranger
// normalization finds a required challenge. It is not a durable outcome and
// consumes no bounded attempt: the receiver's answers cannot exist yet.
const RangerOutcomeAwaitingChallenge = "awaiting_challenge"

// EvaluateRangerToArchivist evaluates and persists one lifecycle-owned
// Ranger-to-Archivist outcome. A required challenge with no input is recorded
// as failed; it is never silently converted into a skip.
func EvaluateRangerToArchivist(strategistRoot, artifactPath, missionID string, input RangerHandoffInput) (RangerHandoffResult, error) {
	return evaluateRangerToArchivist(context.Background(), strategistRoot, artifactPath, missionID, input, nil, missionID)
}

// EvaluateRangerToArchivistWithTelemetry is the CLI completion path. The
// lifecycle result and its terminal telemetry are produced together so a
// successful normalization cannot silently omit the handoff decision.
func EvaluateRangerToArchivistWithTelemetry(ctx context.Context, strategistRoot, artifactPath, missionID string, input RangerHandoffInput, sink telemetry.EventSink, runID string) (RangerHandoffResult, error) {
	return evaluateRangerToArchivist(ctx, strategistRoot, artifactPath, missionID, input, sink, runID)
}

func evaluateRangerToArchivist(ctx context.Context, strategistRoot, artifactPath, missionID string, input RangerHandoffInput, sink telemetry.EventSink, runID string) (RangerHandoffResult, error) {
	tel := newRangerTelemetry(ctx, sink, runID, missionID)
	result, err := recordRangerEvaluation(strategistRoot, artifactPath, missionID, input)
	if err != nil {
		return RangerHandoffResult{}, tel.blocked(err)
	}
	if err := tel.terminal(result.Outcome); err != nil {
		return RangerHandoffResult{}, err
	}
	return result, nil
}

// recordRangerEvaluation prepares, verifies and durably records one attempt.
func recordRangerEvaluation(strategistRoot, artifactPath, missionID string, input RangerHandoffInput) (RangerHandoffResult, error) {
	evaluation, err := prepareRangerEvaluation(strategistRoot, artifactPath, missionID)
	if err != nil {
		return RangerHandoffResult{}, err
	}
	if evaluation.policy.Enabled && evaluation.policy.MaxAttempts > 0 && evaluation.attempt > evaluation.policy.MaxAttempts {
		return RangerHandoffResult{}, fmt.Errorf("ranger_handoff_challenge_exhausted: mission %q reached the maximum of %d attempts", missionID, evaluation.policy.MaxAttempts)
	}
	result := handoff.Verify(evaluation.policy, input.Challenges, input.Ack)
	if !evaluation.policy.Enabled {
		result = handoff.Result{Status: handoff.StatusSkipped, Passed: true}
	}
	outcome, err := appendRangerOutcome(strategistRoot, missionID, evaluation, result)
	if err != nil {
		return RangerHandoffResult{}, err
	}
	return RangerHandoffResult{Outcome: outcome, Result: result, Policy: evaluation.policy}, nil
}

func prepareRangerEvaluation(strategistRoot, artifactPath, missionID string) (rangerEvaluationContext, error) {
	facts, err := handoff.ReadRangerPolicyFacts(artifactPath, missionID)
	if err != nil {
		return rangerEvaluationContext{}, fmt.Errorf("evaluate Ranger-to-Archivist handoff: %w", err)
	}
	policy, err := handoff.RangerToArchivistPolicyForFacts(facts)
	if err != nil {
		return rangerEvaluationContext{}, fmt.Errorf("evaluate Ranger-to-Archivist handoff: %w", err)
	}
	digest, err := handoff.RangerArtifactDigest(artifactPath)
	if err != nil {
		return rangerEvaluationContext{}, fmt.Errorf("evaluate Ranger-to-Archivist handoff: %w", err)
	}
	attempt, err := handoff.NewOutcomeStore(strategistRoot).NextAttemptFor(missionID, handoff.TransitionRangerToArchivist)
	if err != nil {
		return rangerEvaluationContext{}, fmt.Errorf("evaluate Ranger-to-Archivist handoff: %w", err)
	}
	return rangerEvaluationContext{facts: facts, policy: policy, digest: digest, attempt: attempt}, nil
}

func appendRangerOutcome(strategistRoot, missionID string, evaluation rangerEvaluationContext, result handoff.Result) (handoff.Outcome, error) {
	outcomeResult := handoff.OutcomeFailed
	if result.Passed {
		outcomeResult = handoff.OutcomePassed
	}
	if result.Status == handoff.StatusSkipped {
		outcomeResult = handoff.OutcomeSkipped
	}
	outcome, err := handoff.NewOutcomeStore(strategistRoot).Append(handoff.Outcome{
		MissionID: missionID, Transition: handoff.TransitionRangerToArchivist,
		ArtifactDigest: evaluation.digest, Attempt: evaluation.attempt, Result: outcomeResult,
		PolicyID: handoff.PolicyIdentity(evaluation.policy), Required: evaluation.policy.Enabled,
		Provenance:   handoff.RangerPolicyProvenance(evaluation.facts),
		GateObserved: "ranger_normalized", ChallengeStatus: result.Status,
		CriticalFailures: result.CriticalFailures,
	})
	if err != nil {
		return handoff.Outcome{}, fmt.Errorf("evaluate Ranger-to-Archivist handoff: %w", err)
	}
	return outcome, nil
}

// EnsureRangerToArchivistOutcome records the automatic skip after Ranger
// normalization when the typed facts authorize one. A required challenge is
// recorded as nothing at all: no answers can exist yet, so recording a failed
// attempt would only spend one of the bounded attempts. The Archivist boundary
// stays closed (handoff_outcome_missing) until `handoff evaluate-ranger` records
// a real attempt. A missing facts block is intentionally left without an outcome
// so the Archivist boundary reports the precise indeterminate-facts denial
// instead of inventing policy.
func EnsureRangerToArchivistOutcome(strategistRoot, basePath, missionID string) error {
	return EnsureRangerToArchivistOutcomeWithTelemetry(context.Background(), strategistRoot, basePath, missionID, nil, missionID)
}

// EnsureRangerToArchivistOutcomeWithTelemetry is used by mission completion so
// the CLI's terminal path emits the same lifecycle event as explicit evaluation.
// When the challenge is required it emits RangerOutcomeAwaitingChallenge instead.
func EnsureRangerToArchivistOutcomeWithTelemetry(ctx context.Context, strategistRoot, basePath, missionID string, sink telemetry.EventSink, runID string) error {
	artifactPath := filepath.Join(basePath, "pending", missionID+"-analysis.md")
	evaluation, err := prepareRangerEvaluation(strategistRoot, artifactPath, missionID)
	if err != nil {
		err = newRangerTelemetry(ctx, sink, runID, missionID).blocked(err)
		if containsRangerFactsMissing(err) {
			return nil
		}
		return err
	}
	if evaluation.policy.Enabled {
		return newRangerTelemetry(ctx, sink, runID, missionID).awaitingChallenge()
	}
	_, err = EvaluateRangerToArchivistWithTelemetry(ctx, strategistRoot, artifactPath, missionID, RangerHandoffInput{}, sink, runID)
	return err
}

// Package mission contains live mission orchestration adapters.
package mission

import (
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// LiveHandoffInput is the complete input needed at the Archivist-to-Sniper
// transition boundary. The challenge is evaluated and recorded before the
// mission engine is allowed to enter execution.
type LiveHandoffInput struct {
	Policy                    handoff.Policy
	Challenges                []handoff.Challenge
	Ack                       handoff.Acknowledgment
	Attempt                   int
	ConfidenceSummary         *domain.ConfidenceSummary
	PreviousConfidenceSummary *domain.ConfidenceSummary
}

// ArchivistToSniper evaluates and persists one live handoff attempt, then
// advances the mission engine from the independent Approval Gate. A failed
// persistence operation never advances the mission, preserving fail-closed
// authorization semantics.
func ArchivistToSniper(engine *domain.MissionEngine, strategistRoot string, input LiveHandoffInput) (domain.MissionEngineStatus, handoff.Result, error) {
	if engine == nil {
		return domain.MissionEngineStatus{}, handoff.Result{}, fmt.Errorf("live handoff: mission engine is nil")
	}
	status := engine.Status()
	result := handoff.Verify(input.Policy, input.Challenges, input.Ack)
	record := telemetry.ChallengeRecord{
		MissionID:                status.MissionID,
		Transition:               input.Policy.Transition,
		Attempt:                  input.Attempt,
		Timestamp:                time.Now().UTC().Format(time.RFC3339Nano),
		Status:                   result.Status,
		Passed:                   result.Passed,
		MissingRefs:              result.MissingRefs,
		MissingChallenges:        result.MissingChallenges,
		MisclassifiedRefs:        result.MisclassifiedRefs,
		GateMismatch:             result.GateMismatch,
		CounterfactualMismatches: result.CounterfactualMismatches,
		ForbiddenClaimViolations: result.ForbiddenClaimViolations,
		CriticalFailures:         result.CriticalFailures,
	}
	if err := telemetry.AppendHandoffChallenge(telemetry.HandoffChallengeHistoryPath(strategistRoot), record); err != nil {
		return status, result, fmt.Errorf("live handoff: persist attempt: %w", err)
	}
	if err := persistHandoffConfidence(strategistRoot, status.MissionID, input); err != nil {
		return status, result, fmt.Errorf("live handoff: persist confidence: %w", err)
	}

	next, err := engine.SubmitHandoff(domain.HandoffOutcome{
		Attempt:     input.Attempt,
		MaxAttempts: input.Policy.MaxAttempts,
		Passed:      result.Passed,
		NextAction:  result.NextAction,
		Status:      result.Status,
	})
	if err != nil {
		return status, result, fmt.Errorf("live handoff: apply outcome: %w", err)
	}
	return next, result, nil
}

func persistHandoffConfidence(strategistRoot, missionID string, input LiveHandoffInput) error {
	path := telemetry.ConfidenceHistoryPath(strategistRoot)
	producer, err := telemetry.NewConfidenceProducerAdapter(path, telemetry.ConfidenceAgentHandoffChallenge, missionID)
	if err != nil {
		return fmt.Errorf("create confidence producer: %w", err)
	}
	if input.ConfidenceSummary == nil {
		return recordMissingHandoffConfidence(producer, "confidence_summary_not_supplied")
	}
	if err := validateHandoffConfidence(input); err != nil {
		return err
	}
	if input.ConfidenceSummary.MissingRecord {
		return recordMissingHandoffConfidence(producer, "confidence_summary_missing_record")
	}
	return recordHandoffClaims(producer, *input.ConfidenceSummary)
}

func validateHandoffConfidence(input LiveHandoffInput) error {
	if err := domain.ValidateConfidenceSummary(*input.ConfidenceSummary); err != nil {
		return fmt.Errorf("validate confidence summary: %w", err)
	}
	if input.PreviousConfidenceSummary == nil {
		return nil
	}
	if err := domain.CompareConfidenceSummaries(*input.PreviousConfidenceSummary, *input.ConfidenceSummary); err != nil {
		return fmt.Errorf("compare confidence summaries: %w", err)
	}
	return nil
}

func recordMissingHandoffConfidence(producer telemetry.ConfidenceProducerAdapter, reason string) error {
	if err := producer.RecordMissing("archivist_to_sniper", reason); err != nil {
		return fmt.Errorf("record missing confidence summary: %w", err)
	}
	return nil
}

func recordHandoffClaims(producer telemetry.ConfidenceProducerAdapter, summary domain.ConfidenceSummary) error {
	for _, claim := range summary.Claims {
		if _, err := producer.RecordClaim(claim, summary.Evidence); err != nil {
			return fmt.Errorf("record confidence claim %q: %w", claim.ID, err)
		}
	}
	return nil
}

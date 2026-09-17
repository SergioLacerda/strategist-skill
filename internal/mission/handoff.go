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
	Policy     handoff.Policy
	Challenges []handoff.Challenge
	Ack        handoff.Acknowledgment
	Attempt    int
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

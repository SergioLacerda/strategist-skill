package mission

import (
	"context"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/mission/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// emitArchivistHandoff reports the recorded outcome as a bounded event: result,
// attempt, challenge status and the criterion source, never artifact content.
func emitArchivistHandoff(sink telemetry.EventSink, outcome handoff.Outcome) error {
	if sink == nil {
		return nil
	}
	event := telemetry.WithHandoffCriterion(telemetry.NewArchivistToSniperEvent(outcome.MissionID, outcome.Result, outcome.Required, outcome.Attempt, outcome.ChallengeStatus, outcome.CriticalFailures, ""), criterionOf(outcome))
	if err := sink.Emit(context.Background(), event); err != nil {
		return fmt.Errorf("evaluate handoff: emit telemetry: %w", err)
	}
	return nil
}

// recordHandoffConfidence persists the handoff confidence before the outcome,
// so a recording failure authorizes nothing.
func recordHandoffConfidence(strategistRoot, basePath, missionID string, attempt int, input ArchivistHandoffInput) error {
	summary, absentReason := packageConfidenceSummary(refinedDir(basePath, missionID), input.ConfidenceSummary)
	if err := persistHandoffConfidence(strategistRoot, missionID, attempt, summary, absentReason); err != nil {
		return fmt.Errorf("evaluate handoff: handoff_outcome_persist_failed: %w", err)
	}
	return nil
}

// prepareArchivistEvaluation checks the gate was observed, derives the package
// facts and requires the package to be the one the Approval Gate accepted.
func prepareArchivistEvaluation(basePath string, status domain.MissionEngineStatus, input ArchivistHandoffInput) (derivedHandoff, error) {
	if status.State != domain.StateHandoffChallenge {
		return derivedHandoff{}, fmt.Errorf("handoff_gate_not_observed: mission %q is in state %q; the Approval Gate must be accepted before the handoff is evaluated", status.MissionID, status.State)
	}
	derived, err := deriveHandoff(basePath, status.MissionID, input.RiskLevel)
	if err != nil {
		return derivedHandoff{}, fmt.Errorf("evaluate handoff: %w", err)
	}
	if err := requireApprovedPackageDigest(status, derived.digest); err != nil {
		return derivedHandoff{}, fmt.Errorf("evaluate handoff: %w", err)
	}
	return derived, nil
}

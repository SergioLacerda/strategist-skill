package mission

import (
	"context"
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// rangerTelemetry carries what the terminal handoff event needs.
type rangerTelemetry struct {
	ctx   context.Context
	sink  telemetry.EventSink
	runID string
}

func newRangerTelemetry(ctx context.Context, sink telemetry.EventSink, runID, missionID string) rangerTelemetry {
	if runID == "" {
		runID = missionID
	}
	return rangerTelemetry{ctx: ctx, sink: sink, runID: runID}
}

// blocked reports a failed evaluation as an indeterminate, blocked handoff
// event and returns the original error. A missing facts block is deliberately
// silent: the Archivist boundary reports that denial itself.
func (t rangerTelemetry) blocked(err error) error {
	if containsRangerFactsMissing(err) {
		return err
	}
	if telemetryErr := emitRangerHandoffTelemetry(t.ctx, t.sink, t.runID, "indeterminate", false, 0, "blocked", 0, rangerHandoffReason(err)); telemetryErr != nil {
		return fmt.Errorf("evaluate Ranger-to-Archivist handoff: emit telemetry: %w", telemetryErr)
	}
	return err
}

func (t rangerTelemetry) terminal(outcome handoff.Outcome) error {
	if err := emitRangerHandoffTelemetry(t.ctx, t.sink, t.runID, outcome.Result, outcome.Required, outcome.Attempt, outcome.ChallengeStatus, outcome.CriticalFailures, ""); err != nil {
		return fmt.Errorf("evaluate Ranger-to-Archivist handoff: emit telemetry: %w", err)
	}
	return nil
}

func containsRangerFactsMissing(err error) bool {
	return err != nil && strings.Contains(err.Error(), "ranger_handoff_policy_facts_missing")
}

func rangerHandoffReason(err error) string {
	for _, token := range []string{
		"ranger_handoff_policy_facts_missing",
		"ranger_handoff_policy_facts_invalid",
		"ranger_handoff_policy_facts_contradictory",
		"ranger_handoff_policy_facts_ambiguous",
		"handoff_artifact_invalid",
		"ranger_handoff_challenge_exhausted",
		"handoff_outcome_persist_failed",
	} {
		if strings.Contains(err.Error(), token) {
			return token
		}
	}
	return "ranger_handoff_failed"
}

func emitRangerHandoffTelemetry(ctx context.Context, sink telemetry.EventSink, runID, outcome string, required bool, attempt int, status string, criticalFailures int, reason string) error {
	if sink == nil {
		return nil
	}
	if err := sink.Emit(ctx, telemetry.NewRangerToArchivistEvent(runID, outcome, required, attempt, status, criticalFailures, reason)); err != nil {
		return fmt.Errorf("emit Ranger-to-Archivist telemetry: %w", err)
	}
	return nil
}

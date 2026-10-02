package mission

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
)

// RecordArchivistHandoff is the production composition of the
// Archivist-to-Sniper boundary against a loaded mission. It reconciles any
// recorded outcome the mission state has not seen yet, evaluates the handoff,
// and records the evaluation in the engine: a failed attempt returns the mission
// to refinement (a new Approval Gate is then required) and the last allowed
// failure blocks it, while a passed or skipped attempt only records its
// attempt. Entering execution stays a separate event that consumes the outcome.
//
// The caller owns the mission lock, and must save the returned status whenever
// changed is true, even when err is not nil: a reconciliation or a recorded
// outcome is durable state the mission must not lose.
func RecordArchivistHandoff(strategistRoot, basePath string, engine *domain.MissionEngine, input ArchivistHandoffInput) (ArchivistHandoffResult, domain.MissionEngineStatus, bool, error) {
	if engine == nil {
		return ArchivistHandoffResult{}, domain.MissionEngineStatus{}, false, fmt.Errorf("record handoff: mission engine is nil")
	}
	changed, err := reconcileHandoffState(strategistRoot, engine)
	if err != nil {
		return ArchivistHandoffResult{}, engine.Status(), changed, err
	}
	evaluation, err := EvaluateArchivistHandoff(strategistRoot, basePath, engine.Status(), input)
	if err != nil {
		return ArchivistHandoffResult{}, engine.Status(), changed, err
	}
	status, err := engine.RecordHandoffEvaluation(evaluationOf(evaluation.Outcome, evaluation.Result, evaluation.Policy))
	if err != nil {
		return evaluation, engine.Status(), changed, fmt.Errorf("record handoff: %w", err)
	}
	return evaluation, status, true, nil
}

// reconcileHandoffState applies the latest recorded outcome to the mission
// when a crash left it persisted but unrecorded. Without this, a failed outcome
// whose state save was lost would leave the mission at the handoff boundary and
// allow a repaired package to be evaluated without a new Approval Gate.
func reconcileHandoffState(strategistRoot string, engine *domain.MissionEngine) (bool, error) {
	status := engine.Status()
	if status.State != domain.StateHandoffChallenge {
		return false, nil
	}
	store := handoff.NewOutcomeStore(strategistRoot)
	next, err := store.NextAttempt(status.MissionID)
	if err != nil {
		return false, fmt.Errorf("reconcile handoff: %w", err)
	}
	if next-1 <= status.HandoffAttempt {
		return false, nil
	}
	if next-1 != status.HandoffAttempt+1 {
		return false, fmt.Errorf("handoff_outcome_state_mismatch: outcomes reach attempt %d but mission state records attempt %d", next-1, status.HandoffAttempt)
	}
	latest, err := store.Latest(status.MissionID)
	if err != nil {
		return false, fmt.Errorf("reconcile handoff: %w", err)
	}
	policy := handoff.DefaultPolicy()
	if _, err := engine.RecordHandoffEvaluation(evaluationOf(latest, handoff.Result{Status: latest.ChallengeStatus}, policy)); err != nil {
		return false, fmt.Errorf("reconcile handoff: %w", err)
	}
	return true, nil
}

func evaluationOf(outcome handoff.Outcome, result handoff.Result, policy handoff.Policy) domain.HandoffEvaluation {
	nextAction := ""
	if outcome.Result == handoff.OutcomeFailed {
		nextAction = policy.OnFailure
	}
	return domain.HandoffEvaluation{
		Attempt: outcome.Attempt, MaxAttempts: policy.MaxAttempts, Result: outcome.Result,
		Status: result.Status, NextAction: nextAction,
	}
}

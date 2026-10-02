package domain

import "fmt"

// Handoff evaluation results the engine records. They mirror the persisted
// Archivist-to-Sniper outcome results without depending on the handoff package.
const (
	HandoffEvaluationPassed  = "passed"
	HandoffEvaluationFailed  = "failed"
	HandoffEvaluationSkipped = "skipped"
)

// HandoffEvaluation is one already-persisted Archivist-to-Sniper evaluation the
// engine records. It never authorizes execution: entering execution stays a
// separate event that consumes the durable outcome.
type HandoffEvaluation struct {
	Attempt     int
	MaxAttempts int
	Result      string
	Status      string
	NextAction  string
}

// RecordHandoffEvaluation persists the attempt, status and next action of one
// evaluation and applies the only transitions an evaluation may cause: a failed
// attempt returns to refinement, which forces a new Approval Gate acceptance,
// and the last allowed failed attempt blocks the mission. A passed or skipped
// evaluation changes no state. Attempts are sequential, so a replay or a gap is
// rejected, and an evaluation is only accepted at the handoff boundary.
func (e *MissionEngine) RecordHandoffEvaluation(evaluation HandoffEvaluation) (MissionEngineStatus, error) {
	if e == nil {
		return MissionEngineStatus{}, fmt.Errorf("mission engine: engine is nil")
	}
	if err := e.validateHandoffEvaluation(evaluation); err != nil {
		return e.status, err
	}
	next := e.status
	next.HandoffAttempt, next.HandoffStatus, next.HandoffNextAction = evaluation.Attempt, evaluation.Status, evaluation.NextAction
	if evaluation.Result != HandoffEvaluationFailed {
		e.status = next
		return e.status, nil
	}
	return e.applyFailedEvaluation(next, evaluation)
}

func (e *MissionEngine) validateHandoffEvaluation(evaluation HandoffEvaluation) error {
	if e.status.State != StateHandoffChallenge {
		return fmt.Errorf("mission engine: handoff evaluation is not pending from state %q", e.status.State)
	}
	if evaluation.Attempt <= 0 || evaluation.MaxAttempts <= 0 {
		return fmt.Errorf("mission engine: handoff attempt and max attempts must be positive")
	}
	if evaluation.Attempt != e.status.HandoffAttempt+1 {
		return fmt.Errorf("mission engine: handoff attempt %d is not next attempt %d", evaluation.Attempt, e.status.HandoffAttempt+1)
	}
	switch evaluation.Result {
	case HandoffEvaluationPassed, HandoffEvaluationFailed, HandoffEvaluationSkipped:
		return nil
	default:
		return fmt.Errorf("mission engine: unknown handoff evaluation result %q", evaluation.Result)
	}
}

func (e *MissionEngine) applyFailedEvaluation(next MissionEngineStatus, evaluation HandoffEvaluation) (MissionEngineStatus, error) {
	event := MissionEventHandoffFailed
	if evaluation.Attempt >= evaluation.MaxAttempts {
		event = MissionEventHandoffExhausted
	}
	previous := e.status
	next.ApprovalGatePackageDigest = ""
	e.status = next
	status, err := e.submitFSM(event)
	if err != nil {
		e.status = previous
		return previous, err
	}
	return status, nil
}

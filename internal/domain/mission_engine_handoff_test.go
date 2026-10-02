package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func engineAtHandoff(t *testing.T) *MissionEngine {
	t.Helper()
	engine, _, err := StartMission(MissionStartRequest{MissionID: "m-handoff"})
	require.NoError(t, err)
	submitMissionEvents(t, engine, []MissionEngineEvent{
		MissionEventBootstrapDone, MissionEventIntakeDone, MissionEventDiscoveryDone, MissionEventRefinementDone, MissionEventGateApproved,
	})
	return engine
}

func evaluation(attempt int, result string) HandoffEvaluation {
	return HandoffEvaluation{Attempt: attempt, MaxAttempts: 2, Result: result, Status: result, NextAction: "next-" + result}
}

func TestRecordHandoffEvaluationPassedAndSkippedRecordWithoutEnteringExecution(t *testing.T) {
	for _, result := range []string{HandoffEvaluationPassed, HandoffEvaluationSkipped} {
		engine := engineAtHandoff(t)

		status, err := engine.RecordHandoffEvaluation(evaluation(1, result))

		require.NoError(t, err, result)
		assert.Equal(t, StateHandoffChallenge, status.State, "an evaluation never enters execution: %s", result)
		assert.Equal(t, 1, status.HandoffAttempt)
		assert.Equal(t, result, status.HandoffStatus)
		assert.Equal(t, "next-"+result, status.HandoffNextAction)
	}
}

func TestRecordHandoffEvaluationFailureReturnsToRefinementAndTheLastAttemptBlocks(t *testing.T) {
	engine := engineAtHandoff(t)

	status, err := engine.RecordHandoffEvaluation(evaluation(1, HandoffEvaluationFailed))
	require.NoError(t, err)
	assert.Equal(t, StateRefinement, status.State, "a failed handoff forces refinement, hence a new Approval Gate")
	assert.Equal(t, 1, status.HandoffAttempt)

	submitMissionEvents(t, engine, []MissionEngineEvent{MissionEventRefinementDone, MissionEventGateApproved})
	status, err = engine.RecordHandoffEvaluation(evaluation(2, HandoffEvaluationFailed))
	require.NoError(t, err)
	assert.Equal(t, StateBlocked, status.State, "the last allowed attempt exhausts the mission")
	assert.Equal(t, PhaseBlocked, status.Phase)
}

func TestRecordHandoffEvaluationRejectsReplayGapsStateAndUnknownResults(t *testing.T) {
	engine := engineAtHandoff(t)
	before := engine.Status()

	for name, bad := range map[string]HandoffEvaluation{
		"gap":            evaluation(2, HandoffEvaluationPassed),
		"zero attempt":   evaluation(0, HandoffEvaluationPassed),
		"no max":         {Attempt: 1, Result: HandoffEvaluationPassed},
		"unknown result": evaluation(1, "approved"),
	} {
		_, err := engine.RecordHandoffEvaluation(bad)
		require.Error(t, err, name)
		assert.Equal(t, before, engine.Status(), "%s: a rejected evaluation leaves the engine unchanged", name)
	}

	_, err := engine.RecordHandoffEvaluation(evaluation(1, HandoffEvaluationPassed))
	require.NoError(t, err)
	_, err = engine.RecordHandoffEvaluation(evaluation(1, HandoffEvaluationPassed))
	require.Error(t, err, "an attempt cannot be replayed")

	notAtBoundary, _, err := StartMission(MissionStartRequest{MissionID: "m-early"})
	require.NoError(t, err)
	_, err = notAtBoundary.RecordHandoffEvaluation(evaluation(1, HandoffEvaluationPassed))
	require.ErrorContains(t, err, "not pending")

	var nilEngine *MissionEngine
	_, err = nilEngine.RecordHandoffEvaluation(evaluation(1, HandoffEvaluationPassed))
	require.Error(t, err)
}

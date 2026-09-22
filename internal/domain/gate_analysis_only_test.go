package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An accepted package with no documentation_target ends as analysis
// delivered. Before this event the FSM could only record it with a false
// sniper_done (via the handoff challenge) or a false gate_denied.
func TestFSMGateApprovedAnalysisOnlyTerminatesAnalysis(t *testing.T) {
	t.Parallel()
	state := domain.RunStateMachine(domain.StateApprovalGate, []domain.TransitionEvent{domain.EventGateApprovedAnalysisOnly})
	assert.Equal(t, domain.StateDoneAnalysis, state)

	assert.Equal(t, domain.StateHandoffChallenge,
		domain.RunStateMachine(domain.StateApprovalGate, []domain.TransitionEvent{domain.EventGateApproved}),
		"a package with documentation targets still enters the handoff challenge")
	assert.Equal(t, domain.StateDoneAnalysis,
		domain.RunStateMachine(domain.StateApprovalGate, []domain.TransitionEvent{domain.EventGateDenied}),
		"rejection is unchanged")
}

func TestMissionEngineRecordsAnAcceptedAnalysisOnlyMission(t *testing.T) {
	t.Parallel()
	engine, _, err := domain.StartMission(domain.MissionStartRequest{MissionID: "m1"})
	require.NoError(t, err)
	for _, event := range []domain.MissionEngineEvent{
		domain.MissionEventBootstrapDone, domain.MissionEventIntakeDone, domain.MissionEventDiscoveryDone,
		domain.MissionEventRefinementDone, domain.MissionEventGateApprovedAnalysisOnly,
	} {
		_, err := engine.Submit(event)
		require.NoError(t, err, event)
	}
	assert.Equal(t, domain.StateDoneAnalysis, engine.Status().State)

	_, err = engine.Submit(domain.MissionEventSniperDone)
	require.Error(t, err, "nothing may execute after an analysis-only acceptance")
}

// A mission that entered the handoff challenge with nothing for Sniper (for
// example via gate_approved before gate_approved_analysis_only existed) can be
// closed truthfully instead of staying stuck or faking sniper_done.
func TestHandoffNotApplicableEndsAStuckAnalysisOnlyMission(t *testing.T) {
	t.Parallel()
	assert.Equal(t, domain.StateDoneAnalysis,
		domain.RunStateMachine(domain.StateHandoffChallenge, []domain.TransitionEvent{domain.EventHandoffNotApplicable}))

	engine, _, err := domain.StartMission(domain.MissionStartRequest{MissionID: "m1"})
	require.NoError(t, err)
	for _, event := range []domain.MissionEngineEvent{
		domain.MissionEventBootstrapDone, domain.MissionEventIntakeDone, domain.MissionEventDiscoveryDone,
		domain.MissionEventRefinementDone, domain.MissionEventGateApproved, domain.MissionEventHandoffNotApplicable,
	} {
		_, err := engine.Submit(event)
		require.NoError(t, err, event)
	}
	assert.Equal(t, domain.StateDoneAnalysis, engine.Status().State)
	assert.True(t, domain.MissionEventRequiresNoDocumentationTargets(domain.MissionEventHandoffNotApplicable))
	assert.False(t, domain.MissionEventRequiresNoDocumentationTargets(domain.MissionEventHandoffPassed))
}

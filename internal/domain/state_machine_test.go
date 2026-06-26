package domain_test

import (
	"math/rand"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestFSMNominalRoute(t *testing.T) {
	t.Parallel()
	// Init → OpportunityAttack (non-empty) → OpportunityAttack (no opp, empty) → Refinement → ApprovalGate → Execution → DoneDelivery
	policy := domain.DefaultMissionPolicy()
	events := []domain.TransitionEvent{
		domain.EventManifestNonEmpty, // Init → OpportunityAttack
		domain.EventManifestEmpty,    // OpportunityAttack → Refinement (no opportunities)
		domain.EventArchivistTasks,   // Refinement → ApprovalGate
		domain.EventGateApproved,     // ApprovalGate → Execution
		domain.EventSniperDone,       // Execution → DoneDelivery
	}
	state := domain.StateInit
	for _, ev := range events {
		state = domain.NextState(state, ev, policy)
	}
	assert.Equal(t, domain.StateDoneDelivery, state)
}

func TestFSMGateDeniedTerminatesAnalysis(t *testing.T) {
	t.Parallel()
	policy := domain.DefaultMissionPolicy()
	state := domain.RunStateMachine(domain.StateApprovalGate,
		[]domain.TransitionEvent{domain.EventGateDenied},
		policy,
	)
	assert.Equal(t, domain.StateDoneAnalysis, state)
}

func TestOpportunityGateApproved_GoesToExec(t *testing.T) {
	t.Parallel()
	state := domain.RunStateMachine(domain.StateOpportunityAttack,
		[]domain.TransitionEvent{domain.EventManifestNonEmpty, domain.EventGateApproved},
		domain.DefaultMissionPolicy(),
	)
	assert.Equal(t, domain.StateOpportunityExec, state)
}

func TestOpportunityGateDenied_GoesToRefinement(t *testing.T) {
	t.Parallel()
	state := domain.RunStateMachine(domain.StateOpportunityGate,
		[]domain.TransitionEvent{domain.EventGateDenied},
		domain.DefaultMissionPolicy(),
	)
	assert.Equal(t, domain.StateRefinement, state)
}

func TestFSMQuickDrawRoute(t *testing.T) {
	t.Parallel()
	policy := domain.DefaultMissionPolicy()

	// Intent detected → QuickDraw state
	s := domain.NextState(domain.StateInit, domain.EventQuickDrawIntent, policy)
	assert.Equal(t, domain.StateQuickDraw, s)

	// Normalize note → gate
	s = domain.NextState(s, domain.EventManifestNonEmpty, policy)
	assert.Equal(t, domain.StateQuickDrawGate, s)

	// User approves → done
	s = domain.NextState(s, domain.EventQuickDrawApprove, policy)
	assert.Equal(t, domain.StateQuickDrawDone, s)

	// Done is absorbing
	s = domain.NextState(s, domain.EventManifestNonEmpty, policy)
	assert.Equal(t, domain.StateQuickDrawDone, s)
}

func TestFSMQuickDrawDecline(t *testing.T) {
	t.Parallel()
	s := domain.RunStateMachine(domain.StateQuickDrawGate,
		[]domain.TransitionEvent{domain.EventQuickDrawDecline},
		domain.DefaultMissionPolicy(),
	)
	assert.Equal(t, domain.StateQuickDrawDone, s)
}

func TestFSMADRRoute(t *testing.T) {
	t.Parallel()
	policy := domain.DefaultMissionPolicy()

	for _, start := range []domain.MissionState{domain.StateDoneAnalysis, domain.StateDoneDelivery} {
		s := domain.NextState(start, domain.EventADRCriterionMet, policy)
		assert.Equal(t, domain.StateADRGate1, s, "from %s", start)

		s = domain.NextState(s, domain.EventADRApproved, policy)
		assert.Equal(t, domain.StateADRGate2, s)

		s = domain.NextState(s, domain.EventADRApproved, policy)
		assert.Equal(t, domain.StateADRDone, s)
	}
}

func TestFSMADRDeclineAtGate1(t *testing.T) {
	t.Parallel()
	s := domain.RunStateMachine(domain.StateADRGate1,
		[]domain.TransitionEvent{domain.EventADRDeclined},
		domain.DefaultMissionPolicy(),
	)
	assert.Equal(t, domain.StateADRDone, s)
}

func TestFSMRetryTransient(t *testing.T) {
	t.Parallel()
	policy := domain.DefaultMissionPolicy()

	// Transient failure in refinement → retrying
	s := domain.NextState(domain.StateRefinement, domain.EventSlotTransient, policy)
	assert.Equal(t, domain.StateRetrying, s)

	// Retry succeeds (manifest non-empty = slot returned artifact)
	s = domain.NextState(s, domain.EventManifestNonEmpty, policy)
	assert.Equal(t, domain.StateRefinement, s)
}

func TestFSMRetryExhausted(t *testing.T) {
	t.Parallel()
	policy := domain.DefaultMissionPolicy()

	s := domain.NextState(domain.StateRetrying, domain.EventSlotTransient, policy)
	assert.Equal(t, domain.StateBlocked, s)

	s = domain.NextState(domain.StateRetrying, domain.EventSlotPermanent, policy)
	assert.Equal(t, domain.StateBlocked, s)
}

func TestFSMSniperOARoute(t *testing.T) {
	t.Parallel()
	policy := domain.DefaultMissionPolicy()

	// Mid-execution OA surfaces → pause at opportunity gate
	s := domain.NextState(domain.StateExecution, domain.EventSniperOA, policy)
	assert.Equal(t, domain.StateOpportunityGate, s)
}

func TestFSMSafetyPropertyLike(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewSource(31))
	allEvents := []domain.TransitionEvent{
		domain.EventManifestEmpty,
		domain.EventManifestNonEmpty,
		domain.EventGateApproved,
		domain.EventGateDenied,
		domain.EventSniperDone,
		domain.EventArchivistNoTasks,
		domain.EventArchivistTasks,
	}

	for i := 0; i < 400; i++ {
		policy := domain.DefaultMissionPolicy()
		state := domain.StateInit
		seenGateApproved := false
		for j := 0; j < 14; j++ {
			ev := allEvents[rng.Intn(len(allEvents))]
			if ev == domain.EventGateApproved {
				seenGateApproved = true
			}
			state = domain.NextState(state, ev, policy)
			if state == domain.StateExecution {
				assert.True(t, seenGateApproved)
			}
		}
	}
}

func TestFSMCriticalHitRoute(t *testing.T) {
	t.Parallel()
	policy := domain.DefaultMissionPolicy()

	s := domain.NextState(domain.StateInit, domain.EventDirectHitIntent, policy)
	assert.Equal(t, domain.StateDirectGate, s)

	s = domain.NextState(s, domain.EventDirectGateApproved, policy)
	assert.Equal(t, domain.StateDirectExec, s)

	s = domain.NextState(s, domain.EventSniperDone, policy)
	assert.Equal(t, domain.StateDirectDone, s)

	s = domain.NextState(s, domain.EventManifestNonEmpty, policy)
	assert.Equal(t, domain.StateDirectDone, s)
}

func TestFSMCriticalHitDeclinedGoesToDoneAnalysis(t *testing.T) {
	t.Parallel()
	policy := domain.DefaultMissionPolicy()

	s := domain.NextState(domain.StateInit, domain.EventDirectHitIntent, policy)
	assert.Equal(t, domain.StateDirectGate, s)

	s = domain.NextState(s, domain.EventDirectGateDeclined, policy)
	assert.Equal(t, domain.StateDoneAnalysis, s)
}

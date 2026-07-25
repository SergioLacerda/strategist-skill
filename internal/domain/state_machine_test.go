package domain_test

import (
	"math/rand"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestFSMNominalRoute(t *testing.T) {
	t.Parallel()
	// Init -> SideQuestScan (non-empty) -> SideQuestScan (no side quests) -> Refinement -> ApprovalGate -> Execution -> DoneDelivery
	events := []domain.TransitionEvent{
		domain.EventManifestNonEmpty, // Init -> SideQuestScan
		domain.EventManifestEmpty,    // SideQuestScan -> Refinement (no side quests)
		domain.EventArchivistTasks,   // Refinement -> ApprovalGate
		domain.EventGateApproved,     // ApprovalGate -> Execution
		domain.EventSniperDone,       // Execution -> DoneDelivery
	}
	state := domain.StateInit
	for _, ev := range events {
		state = domain.NextState(state, ev)
	}
	assert.Equal(t, domain.StateDoneDelivery, state)
}

func TestFSMGateDeniedTerminatesAnalysis(t *testing.T) {
	t.Parallel()
	state := domain.RunStateMachine(domain.StateApprovalGate,
		[]domain.TransitionEvent{domain.EventGateDenied},
	)
	assert.Equal(t, domain.StateDoneAnalysis, state)
}

func TestFSMGateTimeoutTerminatesAnalysis(t *testing.T) {
	t.Parallel()
	state := domain.RunStateMachine(domain.StateApprovalGate,
		[]domain.TransitionEvent{domain.EventGateTimeout},
	)
	assert.Equal(t, domain.StateDoneAnalysis, state)
}

func TestSideQuestGateApproved_GoesToExec(t *testing.T) {
	t.Parallel()
	state := domain.RunStateMachine(domain.StateSideQuestScan,
		[]domain.TransitionEvent{domain.EventManifestNonEmpty, domain.EventGateApproved},
	)
	assert.Equal(t, domain.StateSideQuestExec, state)
}

func TestSideQuestGateDenied_GoesToRefinement(t *testing.T) {
	t.Parallel()
	state := domain.RunStateMachine(domain.StateSideQuestGate,
		[]domain.TransitionEvent{domain.EventGateDenied},
	)
	assert.Equal(t, domain.StateRefinement, state)
}

func TestFSMQuickDrawRoute(t *testing.T) {
	t.Parallel()

	// Intent detected → QuickDraw state
	s := domain.NextState(domain.StateInit, domain.EventQuickDrawIntent)
	assert.Equal(t, domain.StateQuickDraw, s)

	// Normalize note → gate
	s = domain.NextState(s, domain.EventManifestNonEmpty)
	assert.Equal(t, domain.StateQuickDrawGate, s)

	// User approves → done
	s = domain.NextState(s, domain.EventQuickDrawApprove)
	assert.Equal(t, domain.StateQuickDrawDone, s)

	// Done is absorbing
	s = domain.NextState(s, domain.EventManifestNonEmpty)
	assert.Equal(t, domain.StateQuickDrawDone, s)
}

func TestFSMQuickDrawDecline(t *testing.T) {
	t.Parallel()
	s := domain.RunStateMachine(domain.StateQuickDrawGate,
		[]domain.TransitionEvent{domain.EventQuickDrawDecline},
	)
	assert.Equal(t, domain.StateQuickDrawDone, s)
}

func TestFSMADRRoute(t *testing.T) {
	t.Parallel()

	for _, start := range []domain.MissionState{domain.StateDoneAnalysis, domain.StateDoneDelivery} {
		s := domain.NextState(start, domain.EventADRCriterionMet)
		assert.Equal(t, domain.StateADRGate1, s, "from %s", start)

		s = domain.NextState(s, domain.EventADRApproved)
		assert.Equal(t, domain.StateADRGate2, s)

		s = domain.NextState(s, domain.EventADRApproved)
		assert.Equal(t, domain.StateADRDone, s)
	}
}

func TestFSMADRDeclineAtGate1(t *testing.T) {
	t.Parallel()
	s := domain.RunStateMachine(domain.StateADRGate1,
		[]domain.TransitionEvent{domain.EventADRDeclined},
	)
	assert.Equal(t, domain.StateADRDone, s)
}

func TestFSMRetryTransient(t *testing.T) {
	t.Parallel()

	// Transient failure in refinement preserves refinement retry origin.
	s := domain.NextState(domain.StateRefinement, domain.EventSlotTransient)
	assert.Equal(t, domain.StateRetryingRefinement, s)

	// Retry succeeds (manifest non-empty = slot returned artifact)
	s = domain.NextState(s, domain.EventManifestNonEmpty)
	assert.Equal(t, domain.StateRefinement, s)
}

func TestFSMRetryTransientExecutionPreservesOrigin(t *testing.T) {
	t.Parallel()

	s := domain.NextState(domain.StateExecution, domain.EventSlotTransient)
	assert.Equal(t, domain.StateRetryingExecution, s)

	s = domain.NextState(s, domain.EventManifestNonEmpty)
	assert.Equal(t, domain.StateExecution, s)
}

func TestFSMRetryTransientDirectExecPreservesOrigin(t *testing.T) {
	t.Parallel()

	s := domain.NextState(domain.StateDirectExec, domain.EventSlotTransient)
	assert.Equal(t, domain.StateRetryingDirectExec, s)

	s = domain.NextState(s, domain.EventManifestNonEmpty)
	assert.Equal(t, domain.StateDirectExec, s)
}

func TestFSMRetryExhausted(t *testing.T) {
	t.Parallel()

	for _, state := range []domain.MissionState{
		domain.StateRetrying,
		domain.StateRetryingRefinement,
		domain.StateRetryingExecution,
		domain.StateRetryingDirectExec,
	} {
		s := domain.NextState(state, domain.EventSlotTransient)
		assert.Equal(t, domain.StateBlocked, s, "transient exhaustion from %s", state)

		s = domain.NextState(state, domain.EventSlotPermanent)
		assert.Equal(t, domain.StateBlocked, s, "permanent failure from %s", state)
	}
}

func TestFSMExecutionSideQuestRoute(t *testing.T) {
	t.Parallel()

	// Mid-execution side quest surfaces -> pause at the side quest gate.
	s := domain.NextState(domain.StateExecution, domain.EventSniperSideQuest)
	assert.Equal(t, domain.StateSideQuestGate, s)
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
		assertRandomFSMSequence(t, rng, allEvents)
	}
}

func assertRandomFSMSequence(t *testing.T, rng *rand.Rand, allEvents []domain.TransitionEvent) {
	t.Helper()
	state := domain.StateInit
	seenGateApproved := false
	for j := 0; j < 14; j++ {
		ev := allEvents[rng.Intn(len(allEvents))]
		if ev == domain.EventGateApproved {
			seenGateApproved = true
		}
		state = domain.NextState(state, ev)
		if state == domain.StateExecution {
			assert.True(t, seenGateApproved)
		}
	}
}

func TestFSMCriticalHitRoute(t *testing.T) {
	t.Parallel()

	s := domain.NextState(domain.StateInit, domain.EventDirectHitIntent)
	assert.Equal(t, domain.StateDirectGate, s)

	s = domain.NextState(s, domain.EventDirectGateApproved)
	assert.Equal(t, domain.StateDirectExec, s)

	s = domain.NextState(s, domain.EventSniperDone)
	assert.Equal(t, domain.StateDirectDone, s)

	s = domain.NextState(s, domain.EventManifestNonEmpty)
	assert.Equal(t, domain.StateDirectDone, s)
}

func TestFSMCriticalHitDeclinedGoesToDoneAnalysis(t *testing.T) {
	t.Parallel()

	s := domain.NextState(domain.StateInit, domain.EventDirectHitIntent)
	assert.Equal(t, domain.StateDirectGate, s)

	s = domain.NextState(s, domain.EventDirectGateDeclined)
	assert.Equal(t, domain.StateDoneAnalysis, s)
}

func TestFSMStayBranches(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		start domain.MissionState
		event domain.TransitionEvent
	}{
		{"init unrelated event", domain.StateInit, domain.EventGateApproved},
		{"side quest scan unrelated event", domain.StateSideQuestScan, domain.EventGateApproved},
		{"side quest gate unrelated event", domain.StateSideQuestGate, domain.EventManifestEmpty},
		{"side quest exec unrelated event", domain.StateSideQuestExec, domain.EventGateApproved},
		{"refinement unrelated event", domain.StateRefinement, domain.EventGateApproved},
		{"approval gate unrelated event", domain.StateApprovalGate, domain.EventManifestEmpty},
		{"execution unrelated event", domain.StateExecution, domain.EventGateApproved},
		{"retrying unrelated event", domain.StateRetrying, domain.EventManifestEmpty},
		{"retrying refinement unrelated event", domain.StateRetryingRefinement, domain.EventManifestEmpty},
		{"retrying execution unrelated event", domain.StateRetryingExecution, domain.EventManifestEmpty},
		{"retrying direct exec unrelated event", domain.StateRetryingDirectExec, domain.EventManifestEmpty},
		{"quick draw unrelated event", domain.StateQuickDraw, domain.EventGateApproved},
		{"quick draw gate unrelated event", domain.StateQuickDrawGate, domain.EventManifestEmpty},
		{"adr gate1 unrelated event", domain.StateADRGate1, domain.EventGateApproved},
		{"adr gate2 unrelated event", domain.StateADRGate2, domain.EventGateApproved},
		{"direct gate unrelated event", domain.StateDirectGate, domain.EventGateApproved},
		{"direct exec unrelated event", domain.StateDirectExec, domain.EventGateApproved},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := domain.NextState(tc.start, tc.event)
			assert.Equal(t, tc.start, got, "state should not change for unrelated event")
		})
	}
}

func TestFSMAbsorbingStates(t *testing.T) {
	t.Parallel()
	for _, s := range []domain.MissionState{
		domain.StateQuickDrawDone, domain.StateADRDone, domain.StateBlocked, domain.StateDirectDone,
	} {
		got := domain.NextState(s, domain.EventGateApproved)
		assert.Equal(t, s, got)
	}
}

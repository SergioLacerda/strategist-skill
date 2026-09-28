package mission_test

import (
	"bytes"
	"path/filepath"
	"sync"
	"testing"

	mission "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/filelock"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// newTestCmd returns a bare *cobra.Command with its output captured to a
// private buffer, so concurrent RunStart/RunSubmit calls in the same test do
// not interleave writes to a shared stdout.
func newTestCmd(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	return cmd
}

// lockedLifecycleDeps mirrors lifecycleDeps but wires Lock to the same
// internal/filelock mechanism cmd/strategist/mission_persistence.go's
// production lockMission uses, keyed identically (missions/<id>.json), so
// this test exercises the real serialization path rather than a test double.
func lockedLifecycleDeps(t *testing.T) mission.LifecycleDependencies {
	t.Helper()
	deps := lifecycleDeps(t)
	deps.Lock = func(root, missionID string, fn func() error) error {
		return filelock.WithLock(filepath.Join(root, "missions", missionID+".json"), fn)
	}
	return deps
}

// advanceToApprovalGate drives a fresh mission through the early phase events
// up to StateApprovalGate/PhaseApprovalGate, from which gate_approved and
// gate_denied are two distinct, mutually exclusive valid transitions — the
// scenario ADR-0057 § D2 is about: two concurrent submitters reading the same
// prior state and each believing their own event was accepted.
func advanceToApprovalGate(t *testing.T, deps mission.LifecycleDependencies, root, missionID string) {
	t.Helper()
	require.NoError(t, mission.RunStart(newTestCmd(t), deps, root, missionID, false))
	for _, event := range []string{"bootstrap_done", "intake_done", "discovery_done", "refinement_done"} {
		require.NoError(t, mission.RunSubmit(newTestCmd(t), deps, root, missionID, event, false))
	}
	_, status, err := deps.Load(root, missionID)
	require.NoError(t, err)
	require.Equal(t, domain.PhaseApprovalGate, status.Phase)
}

// TestRunSubmit_ConcurrentCompetingEventsNeverLoseTheWinnersWrite covers
// ADR-0057 § D2's core guarantee: when two sessions concurrently submit
// different, individually valid events for the same mission id, the lock
// ensures they cannot both read the same prior state and each believe their
// event was accepted. Exactly one succeeds; the other is cleanly rejected by
// the FSM (because by the time it acquires the lock, the state has already
// moved past the state its event was valid from) — and the persisted state
// agrees with whichever one actually succeeded. No accepted transition is
// silently overwritten by a losing writer.
func TestRunSubmit_ConcurrentCompetingEventsNeverLoseTheWinnersWrite(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	missionID := "m-gate-race"
	deps := lockedLifecycleDeps(t)
	advanceToApprovalGate(t, deps, root, missionID)

	var wg sync.WaitGroup
	results := make(map[string]error, 2)
	var mu sync.Mutex
	start := make(chan struct{})

	submit := func(event string) {
		defer wg.Done()
		<-start
		err := mission.RunSubmit(newTestCmd(t), deps, root, missionID, event, false)
		mu.Lock()
		results[event] = err
		mu.Unlock()
	}

	wg.Add(2)
	go submit("gate_approved")
	go submit("gate_denied")
	close(start)
	wg.Wait()

	approvedErr := results["gate_approved"]
	deniedErr := results["gate_denied"]

	// Exactly one must succeed and the other must be rejected — never both
	// succeeding (which would mean the second silently overwrote the first's
	// committed state) and never both failing (which would mean the lock
	// deadlocked or corrupted state).
	require.NotEqual(t, approvedErr == nil, deniedErr == nil,
		"expected exactly one of gate_approved/gate_denied to succeed, got approved_err=%v denied_err=%v", approvedErr, deniedErr)

	_, finalStatus, err := deps.Load(root, missionID)
	require.NoError(t, err)

	if approvedErr == nil {
		require.Equal(t, domain.StateHandoffChallenge, finalStatus.State,
			"gate_approved won the race; persisted state must reflect its transition, not be overwritten")
		require.ErrorContains(t, deniedErr, "rejected")
	} else {
		require.Equal(t, domain.StateDoneAnalysis, finalStatus.State,
			"gate_denied won the race; persisted state must reflect its transition, not be overwritten")
		require.ErrorContains(t, approvedErr, "rejected")
	}
}

// TestRunSubmit_ConcurrentIdenticalEventsApplyExactlyOnce covers the simpler
// case: many concurrent submitters racing the *same* event. The lock must
// serialize them so the transition is applied exactly once (by whichever
// submitter acquires the lock first); every other submitter observes the
// already-advanced state and is rejected, rather than each independently
// computing "success" from a stale read and each writing over the last.
func TestRunSubmit_ConcurrentIdenticalEventsApplyExactlyOnce(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	missionID := "m-same-event-race"
	deps := lockedLifecycleDeps(t)
	advanceToApprovalGate(t, deps, root, missionID)

	const attempts = 8
	var wg sync.WaitGroup
	errs := make([]error, attempts)
	start := make(chan struct{})

	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			errs[idx] = mission.RunSubmit(newTestCmd(t), deps, root, missionID, "gate_approved", false)
		}(i)
	}
	close(start)
	wg.Wait()

	successes := 0
	for _, err := range errs {
		if err == nil {
			successes++
		}
	}
	require.Equal(t, 1, successes, "exactly one of %d concurrent identical submits must succeed", attempts)

	_, finalStatus, err := deps.Load(root, missionID)
	require.NoError(t, err)
	require.Equal(t, domain.StateHandoffChallenge, finalStatus.State)
}

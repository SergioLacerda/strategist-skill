package domain_test

import (
	"math/rand"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestFSMAnaliseNeverExecutes(t *testing.T) {
	t.Parallel()
	policy := domain.NewMissionPolicy(domain.MissionModeAnalise)
	state := domain.StateInit
	events := []domain.TransitionEvent{
		domain.EventManifestNonEmpty,
		domain.EventGateApproved,
		domain.EventArchivistTasks,
		domain.EventGateApproved,
		domain.EventSniperDone,
	}
	for _, ev := range events {
		state = domain.NextState(state, ev, policy)
		assert.NotEqual(t, domain.StateExecution, state)
	}
}

func TestOpportunityGatePolicyLocked(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{domain.MissionModeAnalise, domain.MissionModeEntregaRevisada} {
		state := domain.RunStateMachine(domain.StateHousekeeping,
			[]domain.TransitionEvent{domain.EventManifestNonEmpty, domain.EventGateApproved},
			domain.NewMissionPolicy(mode),
		)
		assert.Equal(t, domain.StateRefinement, state)
	}
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
		mode := domain.MissionModeEntregaExecutada
		if rng.Intn(3) == 0 {
			mode = domain.MissionModeAnalise
		} else if rng.Intn(2) == 0 {
			mode = domain.MissionModeEntregaRevisada
		}
		policy := domain.NewMissionPolicy(mode)
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
				assert.True(t, policy.CanExecute)
			}
		}
	}
}

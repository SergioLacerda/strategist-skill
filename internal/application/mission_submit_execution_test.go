package application

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/stretchr/testify/require"
)

func TestCommitMissionExecutionPreservesOutcomeThenClaimsOrder(t *testing.T) {
	order := []string{}
	outcome := handoff.Outcome{MissionID: "m-1"}
	err := CommitMissionExecution(SubmitExecutionCommitRequest{
		Root: "root", BasePath: "base", MissionID: "m-1", Event: domain.MissionEventHandoffSatisfied,
		Outcome: &outcome, EntryPresent: true,
	}, SubmitExecutionCommitPorts{
		PersistOutcome:    func(string, handoff.Outcome) error { order = append(order, "outcome"); return nil },
		RecordEntryClaims: func(string, string, string) error { order = append(order, "entry-claims"); return nil },
		RecordSniperClaims: func(string, string, string, domain.MissionEngineEvent) error {
			order = append(order, "sniper-claims")
			return nil
		},
	})

	require.NoError(t, err)
	require.Equal(t, []string{"outcome", "entry-claims"}, order)
}

func TestRecoverMissionExecutionDoesNotConsumeUnreadyEntry(t *testing.T) {
	called := false
	status := domain.MissionEngineStatus{MissionID: "m-1", State: domain.StateExecution}
	err := RecoverMissionExecution(RecoverExecutionRequest{Root: "root", BasePath: "base", MissionID: "m-1", PersistedStatus: status}, RecoverExecutionPorts{
		Load: func(string, string) (ExecutionRecoverySnapshot, bool, error) {
			return ExecutionRecoverySnapshot{MissionID: "m-1", DesiredStatus: domain.MissionEngineStatus{MissionID: "m-1", State: domain.StateRefinement}}, true, nil
		},
		MarkStateSaved: func(string, ExecutionRecoverySnapshot) error { called = true; return nil },
		RecoverOutcome: func(string, handoff.Outcome) error { called = true; return nil },
		CompleteClaims: func(string, string, string, ExecutionRecoverySnapshot) error { called = true; return nil },
	})

	require.NoError(t, err)
	require.False(t, called)
}

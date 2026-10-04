package application

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestDecideSideQuestLoadsBeforeRecording(t *testing.T) {
	var order []string
	result, err := DecideSideQuest(SideQuestDecisionRequest{MissionID: "m-1"}, func(string, string) (domain.MissionState, error) {
		order = append(order, "load")
		return domain.StateApprovalGate, nil
	}, func(request SideQuestDecisionRequest) (SideQuestDecisionResult, error) {
		order = append(order, "decide")
		require.Equal(t, domain.StateApprovalGate, request.State)
		return SideQuestDecisionResult{MissionID: request.MissionID}, nil
	})

	require.NoError(t, err)
	require.Equal(t, []string{"load", "decide"}, order)
	require.Equal(t, "m-1", result.MissionID)
}

func TestReserveADRTargetLoadsBeforeReservation(t *testing.T) {
	var order []string
	path, err := ReserveADRTarget(ADRTargetRequest{MissionID: "m-1"}, func(string, string) (domain.MissionState, error) {
		order = append(order, "load")
		return domain.StateExecution, nil
	}, func(request ADRTargetRequest, state domain.MissionState) (string, error) {
		order = append(order, "reserve")
		require.Equal(t, domain.StateExecution, state)
		return request.MissionID + ".md", nil
	})

	require.NoError(t, err)
	require.Equal(t, []string{"load", "reserve"}, order)
	require.Equal(t, "m-1.md", path)
}

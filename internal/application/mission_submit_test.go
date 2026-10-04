package application

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestSubmitMissionAppliesTransitionAndApprovalDigest(t *testing.T) {
	engine, _, err := domain.StartMission(domain.MissionStartRequest{MissionID: "m-1"})
	require.NoError(t, err)
	for _, event := range []domain.MissionEngineEvent{
		domain.MissionEventBootstrapDone, domain.MissionEventIntakeDone,
		domain.MissionEventDiscoveryDone, domain.MissionEventRefinementDone,
	} {
		_, err = engine.Submit(event)
		require.NoError(t, err)
	}

	status, err := SubmitMission(engine, SubmitMissionRequest{
		Event: domain.MissionEventGateApproved, GateDigest: "sha256:package",
	})

	require.NoError(t, err)
	require.Equal(t, domain.StateHandoffChallenge, status.State)
	require.Equal(t, "sha256:package", status.ApprovalGatePackageDigest)
}

func TestSubmitMissionRequiresEngineAndEvent(t *testing.T) {
	_, err := SubmitMission(nil, SubmitMissionRequest{Event: domain.MissionEventBootstrapDone})
	require.EqualError(t, err, "mission submit: mission engine is required")

	engine, _, startErr := domain.StartMission(domain.MissionStartRequest{MissionID: "m-1"})
	require.NoError(t, startErr)
	_, err = SubmitMission(engine, SubmitMissionRequest{})
	require.EqualError(t, err, "mission submit: event is required")
}

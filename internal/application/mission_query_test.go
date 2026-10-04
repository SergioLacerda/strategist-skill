package application_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestQueryMissionIsReadOnly(t *testing.T) {
	t.Parallel()

	status := domain.MissionEngineStatus{MissionID: "m-1", Phase: domain.PhaseRefinement, State: domain.StateRefinement, HandoffAttempt: 2}
	got, err := application.QueryMission(status)
	require.NoError(t, err)
	require.Equal(t, application.MissionSummary{MissionID: "m-1", Phase: domain.PhaseRefinement, State: domain.StateRefinement, HandoffAttempt: 2}, got)
}

package mission

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestActivateCriticalHitRoutePersistsIntentAndIsIdempotent(t *testing.T) {
	root := t.TempDir()
	engine, _, err := domain.StartMission(domain.MissionStartRequest{MissionID: "m-route-critical"})
	require.NoError(t, err)
	_, err = engine.Submit(domain.MissionEventBootstrapDone)
	require.NoError(t, err)
	require.NoError(t, Save(root, engine.Status()))
	raw := `{"mission_id":"m-route-critical","request_category":"analysis_move","selected_route":"critical_hit","route_reason":"eligible","route_confidence":0.95,"evidence_state":"explicit","fallback_route":"full_pipeline"}`
	_, err = RecordRouteDecision(root, "m-route-critical", []byte(raw))
	require.NoError(t, err)
	_, persisted, err := Load(root, "m-route-critical")
	require.NoError(t, err)
	require.Equal(t, domain.PhaseIntake, persisted.Phase)
	require.Equal(t, domain.StateInit, persisted.State, "route telemetry alone must not advance mission state")

	status, err := ActivateCriticalHitRoute(root, "m-route-critical", Load, Save)
	require.NoError(t, err)
	require.Equal(t, domain.StateApprovalGate, status.State)
	require.Equal(t, domain.StageShort, status.Stage)
	require.Equal(t, "critical_hit", status.StageFeat)
	require.False(t, status.StageGateApproved)

	replayed, err := ActivateCriticalHitRoute(root, "m-route-critical", Load, Save)
	require.NoError(t, err)
	require.Equal(t, status, replayed)
}

func TestActivateCriticalHitRouteRejectsDoneAnalysisReplay(t *testing.T) {
	root := t.TempDir()
	status := domain.MissionEngineStatus{
		MissionID: "m-route-done", Phase: domain.PhaseDone, State: domain.StateDoneAnalysis,
		Stage: domain.StageShort, StageFeat: "critical_hit", StageCorrelationID: "m-route-done:critical_hit",
		StageGateRequired: true,
	}
	_, err := domain.RestoreMission(status)
	require.NoError(t, err)
	require.NoError(t, Save(root, status))
	_, err = ActivateCriticalHitRoute(root, "m-route-done", Load, Save)
	require.ErrorContains(t, err, "cannot replay")
}

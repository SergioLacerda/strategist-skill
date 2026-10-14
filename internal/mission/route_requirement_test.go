package mission

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireRecordedRouteRejectsMissionWithoutDecision(t *testing.T) {
	err := RequireRecordedRoute(t.TempDir(), boundaryMission)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error=route_decision_missing")
	assert.Contains(t, err.Error(), "strategist mission route")
}

func TestRequireRecordedRouteAcceptsRecordedDecision(t *testing.T) {
	root := t.TempDir()
	recordRoute(t, root, "full_pipeline")

	require.NoError(t, RequireRecordedRoute(root, boundaryMission))
}

func TestRequireRecordedRouteIgnoresAnotherMissionsDecision(t *testing.T) {
	root := t.TempDir()
	recordRoute(t, root, "full_pipeline")

	require.Error(t, RequireRecordedRoute(root, "another-mission"))
}

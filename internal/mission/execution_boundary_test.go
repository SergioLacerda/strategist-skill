package mission

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const boundaryMission = "20260925-boundary"

func handoffStatus() domain.MissionEngineStatus {
	return domain.MissionEngineStatus{MissionID: boundaryMission, Phase: domain.PhaseApprovalGate, State: domain.StateHandoffChallenge}
}

func writeRefined(t *testing.T, base string, files ...string) {
	t.Helper()
	dir := filepath.Join(base, "refined", boundaryMission)
	require.NoError(t, os.MkdirAll(dir, 0o750))
	for _, name := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("x\n"), 0o600))
	}
}

func recordRoute(t *testing.T, root, route string) {
	t.Helper()
	raw := `{"mission_id":"` + boundaryMission + `","request_category":"general","selected_route":"` + route + `","route_reason":"r","route_confidence":0.9,"evidence_state":"requires_discovery","fallback_route":"full_pipeline"}`
	appended, err := RecordRouteDecision(root, boundaryMission, []byte(raw))
	require.NoError(t, err)
	require.True(t, appended)
}

func TestExecutionEntryAllowsCompleteFullPipeline(t *testing.T) {
	root, base := t.TempDir(), t.TempDir()
	recordRoute(t, root, "full_pipeline")
	writeRefined(t, base, "analysis.md", "proposal.md", "design.md", "tasks.md")

	decision, err := EvaluateExecutionEntry(root, base, handoffStatus())

	require.NoError(t, err)
	assert.True(t, decision.Allowed, decision.Error())
}

func TestExecutionEntryBlocksFullPipelineMissingEvidence(t *testing.T) {
	cases := []struct {
		name, phase string
		files       []string
	}{
		{"no discovery", "ranger", nil},
		{"no refinement", "archivist", []string{"analysis.md"}},
		{"no tasks", "archivist", []string{"analysis.md", "proposal.md", "design.md"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, base := t.TempDir(), t.TempDir()
			recordRoute(t, root, "full_pipeline")
			writeRefined(t, base, tc.files...)

			decision, err := EvaluateExecutionEntry(root, base, handoffStatus())

			require.NoError(t, err)
			assert.False(t, decision.Allowed)
			assert.Equal(t, domain.PipelineBypassDetectedReason, decision.Reason)
			assert.Equal(t, tc.phase, decision.ExpectedPhase)
		})
	}
}

// With no recorded Scout decision the strictest regime applies, so a mission
// that skipped Scout's recording cannot use a narrower route by omission.
func TestExecutionEntryDefaultsToStrictestRouteWithoutDecision(t *testing.T) {
	root, base := t.TempDir(), t.TempDir()

	decision, err := EvaluateExecutionEntry(root, base, handoffStatus())

	require.NoError(t, err)
	assert.False(t, decision.Allowed)
	assert.Equal(t, "ranger", decision.ExpectedPhase)
}

func TestExecutionEntryNarrowRouteNeedsOnlyTheApprovedGate(t *testing.T) {
	for _, route := range []string{"implementation_short_route", "critical_hit"} {
		root, base := t.TempDir(), t.TempDir()
		recordRoute(t, root, route)

		decision, err := EvaluateExecutionEntry(root, base, handoffStatus())

		require.NoError(t, err, route)
		assert.True(t, decision.Allowed, route)
	}
}

// The engine only reaches the handoff challenge through an approved gate; from
// any other state the gate evidence is absent and execution entry is blocked.
func TestExecutionEntryBlocksWhenGateWasNotApproved(t *testing.T) {
	root, base := t.TempDir(), t.TempDir()
	recordRoute(t, root, "implementation_short_route")
	writeRefined(t, base, "analysis.md", "proposal.md", "design.md", "tasks.md")
	status := domain.MissionEngineStatus{MissionID: boundaryMission, Phase: domain.PhaseRefinement, State: domain.StateRefinement}

	decision, err := EvaluateExecutionEntry(root, base, status)

	require.NoError(t, err)
	assert.False(t, decision.Allowed)
	assert.Equal(t, "direct_gate", decision.ExpectedPhase)
}

func TestRecordRouteDecisionValidatesAndIsIdempotent(t *testing.T) {
	root := t.TempDir()
	valid := `{"mission_id":"` + boundaryMission + `","request_category":"general","selected_route":"full_pipeline","route_reason":"r","route_confidence":0.9,"evidence_state":"explicit","fallback_route":"full_pipeline"}`

	appended, err := RecordRouteDecision(root, boundaryMission, []byte(valid))
	require.NoError(t, err)
	assert.True(t, appended)
	appended, err = RecordRouteDecision(root, boundaryMission, []byte(valid))
	require.NoError(t, err)
	assert.False(t, appended, "a second decision for the same mission is skipped")

	_, err = RecordRouteDecision(root, "another-mission", []byte(valid))
	require.Error(t, err, "the decision must belong to the mission it is recorded for")
	_, err = RecordRouteDecision(root, boundaryMission, []byte(`{"mission_id":"`+boundaryMission+`","selected_route":"nope"}`))
	require.Error(t, err)
}

package missionview_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/SergioLacerda/strategist-skill/internal/missionview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildKeepsRegistryOrderAndGateSeparate(t *testing.T) {
	reg, err := domain.NewRoleRegistry([]domain.Role{
		{ID: "scout"}, {ID: "ranger", Slot: "discovery", Phase: 1},
		{ID: "archivist", Slot: "refinement", Phase: 2}, {ID: "sniper", Slot: "execution", Phase: 4},
	})
	require.NoError(t, err)
	v := missionview.Build(missionview.Input{Status: domain.MissionEngineStatus{MissionID: "m-1", Phase: domain.PhaseRefinement}, Registry: reg, Levels: []leveling.Record{{MissionID: "m-1", Level: leveling.Level{Role: "ranger", Model: "Sonnet"}}}})

	require.Len(t, v.Journey, 5)
	assert.Equal(t, []string{"scout", "ranger", "archivist", "gate", "sniper"}, journeyIDs(v))
	assert.Equal(t, "gate", v.Journey[3].Kind)
	assert.True(t, v.Confidence.Advisory)
	assert.Equal(t, "latest_record", v.Leveling.Selection)
}

func TestBuildSelectsOnlyRequestedRun(t *testing.T) {
	v := missionview.Build(missionview.Input{Status: domain.MissionEngineStatus{MissionID: "m-1"}, Registry: domain.DefaultRoleRegistry(), Run: "revision-2", Levels: []leveling.Record{
		{MissionID: "m-1", Run: "revision-1", Level: leveling.Level{Role: "ranger", Model: "Old"}},
		{MissionID: "m-1", Run: "revision-2", Level: leveling.Level{Role: "ranger", Model: "New"}},
	}})
	assert.Equal(t, "selected_run", v.Leveling.Selection)
	require.NotNil(t, v.Leveling.Roles[1].Effective)
	assert.Equal(t, "New", v.Leveling.Roles[1].Effective.Model)
}

func TestBuildReportsSecondarySourceFailuresAndRendersDeterministically(t *testing.T) {
	v := missionview.Build(missionview.Input{
		Status: domain.MissionEngineStatus{MissionID: "m-1", Phase: domain.PhaseRefinement, State: domain.StateRefinement}, Registry: domain.DefaultRoleRegistry(),
		ConfidenceError: errors.New("bad confidence"), GateError: errors.New("bad gate"), LevelsError: errors.New("bad levels"),
	})
	assert.Equal(t, missionview.Unavailable, v.Confidence.Availability)
	assert.Equal(t, missionview.Unavailable, v.ApprovalGate.Availability)
	assert.Equal(t, missionview.Unavailable, v.Leveling.Availability)
	require.Len(t, v.Diagnostics, 3)
	var out bytes.Buffer
	require.NoError(t, missionview.RenderHuman(&out, v))
	assert.Contains(t, out.String(), "Confidence (advisory)\n  availability: unavailable")
	assert.Contains(t, out.String(), "mission_view_leveling_unavailable")
}

func TestRenderHumanIncludesGateOutcomeAndEffectiveLevel(t *testing.T) {
	v := missionview.Build(missionview.Input{Status: domain.MissionEngineStatus{MissionID: "m-1"}, Registry: domain.DefaultRoleRegistry(), GateOutcome: "approved", Levels: []leveling.Record{{MissionID: "m-1", Level: leveling.Level{Role: "ranger", Model: "Sonnet", Effort: "high", Source: "host"}}}})
	var out bytes.Buffer
	require.NoError(t, missionview.RenderHuman(&out, v))
	assert.Contains(t, out.String(), "outcome: approved")
	assert.Contains(t, out.String(), "ranger: Sonnet-High source=host")
}

func TestRenderHumanPropagatesWriterFailures(t *testing.T) {
	v := missionview.Build(missionview.Input{Status: domain.MissionEngineStatus{MissionID: "m-1"}, Registry: domain.DefaultRoleRegistry(), GateOutcome: "approved", Levels: []leveling.Record{{MissionID: "m-1", Level: leveling.Level{Role: "ranger", Model: "Sonnet"}}}})
	for failAt := 1; failAt <= 14; failAt++ {
		err := missionview.RenderHuman(&failingWriter{failAt: failAt}, v)
		assert.Error(t, err, "failAt=%d", failAt)
	}
}

type failingWriter struct{ failAt, calls int }

func (w *failingWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.calls == w.failAt {
		return 0, errors.New("write failed")
	}
	return len(p), nil
}

func journeyIDs(v missionview.View) []string {
	ids := make([]string, len(v.Journey))
	for i, step := range v.Journey {
		ids[i] = step.ID
	}
	return ids
}

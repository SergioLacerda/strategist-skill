package missionview_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/SergioLacerda/strategist-skill/internal/missionview"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
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

// TestBuild_TokenUsageNotApplicableWhenNoneReported covers F-T2 (ADR-0057 §
// design.md task 3.3): a mission with no reported usage is NotApplicable,
// not Unavailable — nothing has gone wrong, most missions never call
// `mission report-usage`.
func TestBuild_TokenUsageNotApplicableWhenNoneReported(t *testing.T) {
	v := missionview.Build(missionview.Input{
		Status:   domain.MissionEngineStatus{MissionID: "m-1"},
		Registry: domain.DefaultRoleRegistry(),
	})
	assert.Equal(t, missionview.NotApplicable, v.TokenUsage.Availability)
	assert.Zero(t, v.TokenUsage.TotalTokensIn)
	assert.Zero(t, v.TokenUsage.TotalTokensOut)
	assert.Empty(t, v.TokenUsage.Records)
}

// TestBuild_TokenUsageSumsReportedRecords covers the same task: two reported
// records show both individually and summed, and the declared budget tier is
// carried through verbatim.
func TestBuild_TokenUsageSumsReportedRecords(t *testing.T) {
	records := []telemetry.MissionTokenUsageRecord{
		{MissionID: "m-1", TokensIn: 100, TokensOut: 50, Source: telemetry.MissionUsageSourceAgentReport, ReportedAt: "2026-09-27T10:00:00Z"},
		{MissionID: "m-1", TokensIn: 200, TokensOut: 75, Source: telemetry.MissionUsageSourceAgentReport, ReportedAt: "2026-09-27T11:00:00Z"},
	}
	v := missionview.Build(missionview.Input{
		Status:              domain.MissionEngineStatus{MissionID: "m-1"},
		Registry:            domain.DefaultRoleRegistry(),
		TokenUsage:          records,
		DeclaredTokenBudget: "high",
	})
	assert.Equal(t, missionview.Available, v.TokenUsage.Availability)
	assert.Equal(t, "high", v.TokenUsage.DeclaredTokenBudget)
	assert.Equal(t, int64(300), v.TokenUsage.TotalTokensIn)
	assert.Equal(t, int64(125), v.TokenUsage.TotalTokensOut)
	require.Len(t, v.TokenUsage.Records, 2)
}

// TestBuild_TokenUsageErrorIsUnavailableWithDiagnostic covers the read-error
// path, mirroring the existing Confidence/Gate/Levels error handling this
// package already tests.
func TestBuild_TokenUsageErrorIsUnavailableWithDiagnostic(t *testing.T) {
	v := missionview.Build(missionview.Input{
		Status:          domain.MissionEngineStatus{MissionID: "m-1"},
		Registry:        domain.DefaultRoleRegistry(),
		TokenUsageError: errors.New("token usage read failed"),
	})
	assert.Equal(t, missionview.Unavailable, v.TokenUsage.Availability)
	found := false
	for _, d := range v.Diagnostics {
		if d.Reason == missionview.ReasonMissionViewTokenUsageUnavailable {
			found = true
		}
	}
	assert.True(t, found, "expected a token-usage-unavailable diagnostic")
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
	assert.Contains(t, out.String(), "ranger: Sonnet-High source=host model_source=")
}

func TestBuildAndRenderHumanExposeProviderBinding(t *testing.T) {
	v := missionview.Build(missionview.Input{Status: domain.MissionEngineStatus{MissionID: "m-1"}, Registry: domain.DefaultRoleRegistry(), SlotProviders: map[string]string{"discovery": "brainstorming"}})
	var out bytes.Buffer
	require.NoError(t, missionview.RenderHuman(&out, v))
	assert.Contains(t, out.String(), "ranger (role) provider=brainstorming")
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

// A fresh mission has neither a calibrated confidence sample nor role-level
// records; both are reported explicitly instead of as "available".
func TestBuildReportsNoSampleAndUnknownForAFreshMission(t *testing.T) {
	v := missionview.Build(missionview.Input{
		Status:   domain.MissionEngineStatus{MissionID: "m1", Phase: domain.PhaseBootstrap, State: domain.StateInit},
		Registry: domain.DefaultRoleRegistry(),
	})
	assert.Equal(t, missionview.NoSample, v.Confidence.Availability)
	assert.Equal(t, missionview.Unknown, v.Leveling.Availability)

	var out strings.Builder
	require.NoError(t, missionview.RenderHuman(&out, v))
	assert.Contains(t, out.String(), "availability: no_sample")
	assert.Contains(t, out.String(), "availability: unknown")
}

package missionview

import (
	"bytes"
	"errors"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failWriter struct {
	failAt int
	count  int
}

func (f *failWriter) Write(p []byte) (n int, err error) {
	f.count++
	if f.count >= f.failAt {
		return 0, errors.New("write failure")
	}
	return len(p), nil
}

func TestBuildMissionViewEdgeCases(t *testing.T) {
	reg := domain.DefaultRoleRegistry()

	// 1. Errors on secondary sources
	inErrors := Input{
		Status:          domain.MissionEngineStatus{MissionID: "m1", Phase: "discovery", State: "active"},
		Registry:        reg,
		ConfidenceError: errors.New("confidence error"),
		GateError:       errors.New("gate error"),
		LevelsError:     errors.New("levels error"),
		Run:             "rev1",
	}
	viewErr := Build(inErrors)
	assert.Equal(t, Unavailable, viewErr.Confidence.Availability)
	assert.Equal(t, Unavailable, viewErr.ApprovalGate.Availability)
	assert.Equal(t, Unavailable, viewErr.Leveling.Availability)
	assert.Equal(t, "selected_run", viewErr.Leveling.Selection)
	assert.Len(t, viewErr.Diagnostics, 3)

	// 2. Empty gate outcome (NotApplicable) and empty levels (Unknown availability)
	inEmpty := Input{
		Status:   domain.MissionEngineStatus{MissionID: "m2", Phase: "execution", State: "completed"},
		Registry: reg,
		Levels:   []leveling.Record{},
	}
	viewEmpty := Build(inEmpty)
	assert.Equal(t, NotApplicable, viewEmpty.ApprovalGate.Availability)
	assert.Equal(t, Unknown, viewEmpty.Leveling.Availability)
	assert.Equal(t, "latest_record", viewEmpty.Leveling.Selection)

	// 3. Effective record selection by run
	recRun1 := leveling.Record{MissionID: "m3", Run: "run1", Level: leveling.Level{Role: "ranger", Model: "gpt-4"}}
	recRun2 := leveling.Record{MissionID: "m3", Run: "run2", Level: leveling.Level{Role: "ranger", Model: "o3-mini"}}
	inRun := Input{
		Status:   domain.MissionEngineStatus{MissionID: "m3", Phase: "execution", State: "active"},
		Registry: reg,
		Levels:   []leveling.Record{recRun1, recRun2},
		Run:      "run1",
	}
	viewRun := Build(inRun)
	assert.NotEmpty(t, viewRun.Leveling.Roles)
	for _, r := range viewRun.Leveling.Roles {
		if r.Role == "ranger" {
			require.NotNil(t, r.Effective)
			assert.Equal(t, "gpt-4", r.Effective.Model)
		}
	}
}

func TestRenderHumanFullAndWriterErrors(t *testing.T) {

	v := View{
		MissionID: "m1",
		Lifecycle: Lifecycle{Phase: "discovery", State: "active"},
		Journey: []JourneyEntry{
			{Kind: "role", ID: "ranger", Phase: 1, Provider: "CODEX"},
			{Kind: "gate", ID: "gate", Phase: 2},
		},
		Confidence:   ConfidenceSection{Availability: Available, Advisory: true},
		ApprovalGate: GateSection{Availability: Available, Outcome: "accepted"},
		Leveling: LevelingSection{
			Availability: Available,
			Selection:    "latest_record",
			Roles: []LevelingRole{
				{Role: "ranger", Effective: &leveling.Record{Level: leveling.Level{Role: "ranger", Model: "gpt-4", Source: "policy"}}},
				{Role: "archivist", Effective: nil},
			},
		},
		Diagnostics: []Diagnostic{
			{Reason: ReasonMissionViewConfidenceUnavailable, Action: "inspect confidence ledger"},
		},
	}

	var buf bytes.Buffer
	require.NoError(t, RenderHuman(&buf, v))
	rendered := buf.String()
	assert.Contains(t, rendered, "provider=CODEX")
	assert.Contains(t, rendered, "outcome: accepted")
	assert.Contains(t, rendered, "archivist: unknown")
	assert.Contains(t, rendered, "inspect confidence ledger")

	// Writer failure paths
	for i := 1; i <= 10; i++ {
		fw := &failWriter{failAt: i}
		_ = RenderHuman(fw, v)
	}
}

func TestConfidenceAvailability(t *testing.T) {
	noSampleReview := telemetry.ConfidenceGateReview{}
	assert.Equal(t, NoSample, confidenceAvailability(noSampleReview))

	sampleReview := telemetry.ConfidenceGateReview{
		Metrics: telemetry.ConfidenceMetrics{SampleSize: 5},
	}
	assert.Equal(t, Available, confidenceAvailability(sampleReview))
}

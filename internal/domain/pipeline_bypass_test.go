package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestEvaluatePipelineBypass_BlocksWithoutDiscovery(t *testing.T) {
	t.Parallel()
	decision := domain.EvaluatePipelineBypass(domain.PipelineEvidence{
		BasePath:        ".analysis",
		MissionID:       "m-readme",
		AttemptedAction: "edit readme.md directly",
	})

	assert.False(t, decision.Allowed)
	assert.Equal(t, domain.PipelineBypassDetectedReason, decision.Reason)
	assert.Equal(t, "ranger", decision.ExpectedPhase)
	assert.Equal(t, ".analysis/refined/m-readme/analysis.md", decision.MissingEvidence)
	assert.Contains(t, decision.ResumeHint, "Ranger")
}

func TestEvaluatePipelineBypass_BlocksWithoutRefinement(t *testing.T) {
	t.Parallel()
	decision := domain.EvaluatePipelineBypass(domain.PipelineEvidence{
		BasePath:         ".analysis",
		MissionID:        "m-readme",
		AttemptedAction:  "edit readme.md directly",
		DiscoveryPresent: true,
	})

	assert.False(t, decision.Allowed)
	assert.Equal(t, "archivist", decision.ExpectedPhase)
	assert.Equal(t, ".analysis/refined/m-readme/", decision.MissingEvidence)
	assert.Contains(t, decision.ResumeHint, "Archivist")
}

func TestEvaluatePipelineBypass_BlocksWithoutTasksOrGate(t *testing.T) {
	t.Parallel()
	noTasks := domain.EvaluatePipelineBypass(domain.PipelineEvidence{
		BasePath:          ".analysis",
		MissionID:         "m-readme",
		AttemptedAction:   "edit readme.md directly",
		DiscoveryPresent:  true,
		RefinementPresent: true,
	})
	assert.False(t, noTasks.Allowed)
	assert.Equal(t, "archivist", noTasks.ExpectedPhase)
	assert.Equal(t, ".analysis/refined/m-readme/tasks.md", noTasks.MissingEvidence)

	noGate := domain.EvaluatePipelineBypass(domain.PipelineEvidence{
		BasePath:          ".analysis",
		MissionID:         "m-readme",
		AttemptedAction:   "edit readme.md directly",
		DiscoveryPresent:  true,
		RefinementPresent: true,
		TasksPresent:      true,
	})
	assert.False(t, noGate.Allowed)
	assert.Equal(t, "approval_gate", noGate.ExpectedPhase)
	assert.Equal(t, "approval_gate:approved", noGate.MissingEvidence)
}

func TestEvaluatePipelineBypass_AllowsMainPipelineAfterGate(t *testing.T) {
	t.Parallel()
	decision := domain.EvaluatePipelineBypass(domain.PipelineEvidence{
		BasePath:          ".analysis",
		MissionID:         "m-readme",
		AttemptedAction:   "edit readme.md directly",
		DiscoveryPresent:  true,
		RefinementPresent: true,
		TasksPresent:      true,
		GatePresented:     true,
		GateApproved:      true,
	})

	assert.True(t, decision.Allowed)
	assert.Empty(t, decision.Reason)
}

func TestEvaluatePipelineBypass_QuickDrawCompatibility(t *testing.T) {
	t.Parallel()
	blocked := domain.EvaluatePipelineBypass(domain.PipelineEvidence{
		Route:           domain.MissionRouteQuickDraw,
		BasePath:        ".analysis",
		MissionID:       "m-note",
		AttemptedAction: "append quick draw note",
	})
	assert.False(t, blocked.Allowed)
	assert.Equal(t, "quick_draw_gate", blocked.ExpectedPhase)
	assert.Contains(t, blocked.MissingEvidence, "quick_draw_gate:approved")

	allowed := domain.EvaluatePipelineBypass(domain.PipelineEvidence{
		Route:              domain.MissionRouteQuickDraw,
		BasePath:           ".analysis",
		MissionID:          "m-note",
		AttemptedAction:    "append quick draw note",
		QuickDrawPresented: true,
		QuickDrawApproved:  true,
	})
	assert.True(t, allowed.Allowed)
}

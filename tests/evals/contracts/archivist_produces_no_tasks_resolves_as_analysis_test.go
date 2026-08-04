//go:build eval

// Scenario B4: Archivist producing no tasks resolves the mission as
// analysis-only, never proceeding toward execution.
package contracts_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/eval"
	"github.com/stretchr/testify/assert"
)

func TestB4_ArchivistProducesNoTasksResolvesAsAnalysis(t *testing.T) {
	res := eval.RunScenario(eval.Scenario{
		ID:          "archivist-produces-no-tasks-resolves-as-analysis",
		Description: "archivist_done_no_tasks transitions REFINEMENT to DONE_ANALYSIS, bypassing the approval gate entirely",
		Input: eval.Input{
			Target: eval.TargetStateMachine,
			Params: map[string]any{
				"start":  "REFINEMENT",
				"events": []any{"archivist_done_no_tasks"},
			},
		},
		Expected: eval.Expected{State: "DONE_ANALYSIS"},
	})
	assert.True(t, res.Passed, "violations: %+v", res.Violations)
}

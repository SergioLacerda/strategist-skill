//go:build eval

// Scenario D1: a Ranger discovery artifact has the required frontmatter and
// section shape. This is a fixture-based check, not a live Ranger run —
// Ranger is agent-embodied, not a Go-callable function, so this test
// validates artifact shape against a hand-authored golden fixture rather
// than a real path-placement check. See
// .analysis/archived/20260804-eval-fake-provider-adr.md (DEC-2) for why this
// is fixture-based content assertion rather than a FakeProvider mock.
package contracts_test

import (
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/eval"
	"github.com/stretchr/testify/assert"
)

func TestD1_RangerArtifactShapeValid(t *testing.T) {
	path := filepath.Join(repoRoot(t), "tests", "evals", "fixtures", "ranger-analysis-example.md")
	res := eval.RunScenario(eval.Scenario{
		ID:          "ranger-artifact-shape-valid",
		Description: "a Ranger analysis artifact fixture has correct frontmatter and all seven required sections",
		Input: eval.Input{
			Target: eval.TargetArtifactCheck,
			Params: map[string]any{
				"path": path,
			},
		},
		Assertions: []eval.Assertion{
			{Type: eval.AssertRegex, Value: `mission_id: \d{8}-`},
			{Type: eval.AssertContains, Value: "mission_status: ranger_pending"},
			{
				Type: eval.AssertRequiredSections,
				Value: "Mission Objective,Known Facts,Uncertainties,Affected Scope," +
					"Side Quests,Scope Observations,Recommended Refinement Focus",
			},
			{Type: eval.AssertSourceCitations, Value: "1"},
		},
	})
	assert.True(t, res.Passed, "violations: %+v", res.Violations)
}

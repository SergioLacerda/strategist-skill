//go:build eval

// Scenario D4: a Critical Hit closure-move completion report has the four
// required fields from 11-critical-hit.md § Closure Requirements. This is a
// fixture-based check, not a live closure move — Critical Hit's move is
// agent-embodied (Sniper native role), not a Go-callable function, so this
// test validates report shape against a hand-authored golden fixture rather
// than exercising a real pending/refined -> done/ move. See
// .analysis/archived/20260804-eval-fake-provider-adr.md (DEC-2).
package contracts_test

import (
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/eval"
	"github.com/stretchr/testify/assert"
)

func TestD4_CriticalHitClosureReportShapeValid(t *testing.T) {
	path := filepath.Join(repoRoot(t), "tests", "evals", "fixtures", "closure-completion-report-example.md")
	res := eval.RunScenario(eval.Scenario{
		ID:          "critical-hit-closure-report-shape-valid",
		Description: "a Critical Hit closure completion report fixture has all four required fields",
		Input: eval.Input{
			Target: eval.TargetArtifactCheck,
			Params: map[string]any{
				"path": path,
			},
		},
		Assertions: []eval.Assertion{
			{
				Type:  eval.AssertRequiredSections,
				Value: "What Was Completed,Evidence Supplied,Checks Run,Unresolved Residuals",
			},
			{Type: eval.AssertNotContains, Value: "git commit"},
			{Type: eval.AssertForbiddenToolCall, Value: "git push"},
		},
	})
	assert.True(t, res.Passed, "violations: %+v", res.Violations)
}

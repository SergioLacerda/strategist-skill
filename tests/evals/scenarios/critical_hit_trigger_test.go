//go:build eval

// Group D: Critical Hit trigger evaluation — a plain-move or closure-move
// request is only allowed when it satisfies the all_of/none_of conditions in
// contracts/machine/critical-hit.yaml#trigger_conditions.
// internal/domain.EvaluateCriticalHit is the deterministic enforcement
// mechanism behind this rule (see internal/domain/critical_hit_trigger.go).
package scenarios_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/eval"
	"github.com/stretchr/testify/assert"
)

func TestD1_CriticalHit_ValidPlainMoveAllowed(t *testing.T) {
	res := eval.RunScenario(eval.Scenario{
		ID:          "critical-hit-valid-plain-move-allowed",
		Description: "a low-risk, ≤5-file, .md-only move between analysis folders is allowed",
		Input: eval.Input{
			Target: eval.TargetCriticalHitTrigger,
			Params: map[string]any{
				"mode":        "plain",
				"task_type":   "analysis_move",
				"source_path": ".analysis/pending/foo-analysis.md",
				"target_path": ".analysis/archived/foo-analysis.md",
				"base_path":   ".analysis",
				"file_types":  []any{".md"},
				"risk_level":  "low",
				"file_count":  1,
			},
		},
		Expected: eval.Expected{Status: "allowed"},
	})
	assert.True(t, res.Passed, "violations: %+v", res.Violations)
}

func TestD1_CriticalHit_ValidClosureMoveAllowed(t *testing.T) {
	res := eval.RunScenario(eval.Scenario{
		ID:          "critical-hit-valid-closure-move-allowed",
		Description: "a closure move with an explicit completion claim and supplied evidence is allowed",
		Input: eval.Input{
			Target: eval.TargetCriticalHitTrigger,
			Params: map[string]any{
				"mode":                      "closure",
				"task_type":                 "analysis_move",
				"source_path":               ".analysis/refined/foo",
				"target_path":               ".analysis/done/foo",
				"base_path":                 ".analysis",
				"explicit_completion_claim": true,
				"evidence_summary_present":  true,
			},
		},
		Expected: eval.Expected{Status: "allowed"},
	})
	assert.True(t, res.Passed, "violations: %+v", res.Violations)
}

func TestD1_CriticalHit_ConditionsNotMetFallsBackToMainMission(t *testing.T) {
	res := eval.RunScenario(eval.Scenario{
		ID:          "critical-hit-conditions-not-met-blocked",
		Description: "a request with no completion claim or evidence for a closure move is blocked, falling back to main_mission",
		Input: eval.Input{
			Target: eval.TargetCriticalHitTrigger,
			Params: map[string]any{
				"mode":        "closure",
				"task_type":   "analysis_move",
				"source_path": ".analysis/refined/foo",
				"target_path": ".analysis/done/foo",
				"base_path":   ".analysis",
			},
		},
		Expected: eval.Expected{Status: "blocked", Reason: "conditions_not_met"},
	})
	assert.True(t, res.Passed, "violations: %+v", res.Violations)
}

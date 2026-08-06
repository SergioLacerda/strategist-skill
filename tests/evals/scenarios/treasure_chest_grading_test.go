//go:build eval

// Group C (continued): treasure-chest grading — a chest's optional grading
// fields (source_grade/reuse_value/implementation_status) must be one of
// their enumerated values when set, and a jewel's trust tier may never
// exceed its parent chest's trust tier. internal/domain.ValidateChestGrade
// and internal/domain.ValidateJewelTrust are the deterministic enforcement
// mechanisms behind these rules (see internal/domain/chest_grade.go,
// internal/domain/jewel_grade.go). Distinct from
// treasure_chest_scope_filter_test.go, which covers chest *consultation
// scope* filtering, not grading.
package scenarios_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/eval"
	"github.com/stretchr/testify/assert"
)

func TestC2_ChestGrade_ValidFieldsAllowed(t *testing.T) {
	res := eval.RunScenario(eval.Scenario{
		ID:          "chest-grade-valid-fields-allowed",
		Description: "a chest grade with all-enumerated field values passes validation",
		Input: eval.Input{
			Target: eval.TargetChestGrade,
			Params: map[string]any{
				"chest_id":              "runbooks",
				"source_grade":          "A",
				"reuse_value":           "high",
				"implementation_status": "implemented",
			},
		},
		Expected: eval.Expected{Status: "allowed"},
	})
	assert.True(t, res.Passed, "violations: %+v", res.Violations)
}

func TestC2_ChestGrade_InvalidSourceGradeBlocked(t *testing.T) {
	res := eval.RunScenario(eval.Scenario{
		ID:          "chest-grade-invalid-source-grade-blocked",
		Description: "a chest grade with an out-of-enum source_grade is rejected",
		Input: eval.Input{
			Target: eval.TargetChestGrade,
			Params: map[string]any{
				"chest_id":     "runbooks",
				"source_grade": "Z",
			},
		},
		Expected: eval.Expected{
			Status: "blocked",
			Reason: "must be one of A, B, C",
		},
	})
	assert.True(t, res.Passed, "violations: %+v", res.Violations)
}

func TestC2_JewelTrust_WithinChestTierAllowed(t *testing.T) {
	res := eval.RunScenario(eval.Scenario{
		ID:          "jewel-trust-within-chest-tier-allowed",
		Description: "a jewel at the same trust tier as its parent chest is allowed",
		Input: eval.Input{
			Target: eval.TargetJewelTrust,
			Params: map[string]any{
				"jewel_id":         "jewel-001",
				"jewel_trust":      "T2",
				"chest_trust_tier": "T2",
			},
		},
		Expected: eval.Expected{Status: "allowed"},
	})
	assert.True(t, res.Passed, "violations: %+v", res.Violations)
}

func TestC2_JewelTrust_ExceedsChestTierBlocked(t *testing.T) {
	res := eval.RunScenario(eval.Scenario{
		ID:          "jewel-trust-exceeds-chest-tier-blocked",
		Description: "a jewel claiming a more-trusted tier (T0) than its T2 parent chest is rejected",
		Input: eval.Input{
			Target: eval.TargetJewelTrust,
			Params: map[string]any{
				"jewel_id":         "jewel-002",
				"jewel_trust":      "T0",
				"chest_trust_tier": "T2",
			},
		},
		Expected: eval.Expected{
			Status: "blocked",
			Reason: "exceeds parent chest's trust tier",
		},
	})
	assert.True(t, res.Passed, "violations: %+v", res.Violations)
}

package runbook

import "testing"

func runbookWithChecks() Runbook {
	return Runbook{
		RunbookID: "example",
		Checks: []Check{
			{ID: "reproduced-locally", Level: LevelMandatory},
			{ID: "changelog-reviewed", Level: LevelRecommended},
			{ID: "rollback-plan", Level: LevelMandatory},
		},
	}
}

func TestValidateCompletion_PassesWhenMandatoryChecksSatisfied(t *testing.T) {
	t.Parallel()
	satisfied := map[string]bool{"reproduced-locally": true, "rollback-plan": true}
	result := ValidateCompletion(runbookWithChecks(), satisfied, nil)
	if !result.Passed {
		t.Fatalf("expected Passed=true, got violations: %v", result.Violations)
	}
}

func TestValidateCompletion_BlocksOnUnsatisfiedMandatoryWithoutJustification(t *testing.T) {
	t.Parallel()
	satisfied := map[string]bool{"reproduced-locally": true}
	result := ValidateCompletion(runbookWithChecks(), satisfied, nil)
	if result.Passed {
		t.Fatal("expected Passed=false: rollback-plan is mandatory, unsatisfied, and unjustified")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected exactly 1 violation, got %v", result.Violations)
	}
}

func TestValidateCompletion_JustifiedMandatoryCheckPassesAsException(t *testing.T) {
	t.Parallel()
	satisfied := map[string]bool{"reproduced-locally": true}
	justifications := map[string]string{"rollback-plan": "no rollback needed, additive-only change"}
	result := ValidateCompletion(runbookWithChecks(), satisfied, justifications)
	if !result.Passed {
		t.Fatalf("expected Passed=true with justification, got violations: %v", result.Violations)
	}
	if len(result.Exceptions) != 1 {
		t.Fatalf("expected 1 recorded exception, got %v", result.Exceptions)
	}
}

func TestValidateCompletion_RecommendedChecksNeverBlock(t *testing.T) {
	t.Parallel()
	satisfied := map[string]bool{"reproduced-locally": true, "rollback-plan": true}
	result := ValidateCompletion(runbookWithChecks(), satisfied, nil)
	if !result.Passed {
		t.Fatalf("expected Passed=true regardless of changelog-reviewed's state, got: %v", result.Violations)
	}
}

func TestValidateCompletion_MissingFromSatisfiedMapTreatedAsUnsatisfied(t *testing.T) {
	t.Parallel()
	result := ValidateCompletion(runbookWithChecks(), map[string]bool{}, map[string]string{})
	if result.Passed {
		t.Fatal("expected Passed=false when mandatory checks are absent from the satisfied map")
	}
}

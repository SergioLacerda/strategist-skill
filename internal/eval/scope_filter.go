package eval

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
)

func runScopeFilterScenario(s Scenario, res *ScenarioResult) {
	rows, value := parseScopeFilterParams(s.Input.Params)
	filtered := treasure.FilterRowsByScope(rows, value)
	actualIDs := make([]string, 0, len(filtered))
	for _, r := range filtered {
		actualIDs = append(actualIDs, r.ID)
	}

	if s.Expected.IDs != nil {
		compareIDSets(res, s.Expected.IDs, actualIDs)
	}
	applyScopeAssertions(s.Assertions, actualIDs, res)
}

func parseScopeFilterParams(params map[string]any) ([]treasure.StatusRow, string) {
	rowsRaw, ok := params["rows"].([]any)
	if !ok {
		return nil, paramString(params, "value")
	}
	rows := make([]treasure.StatusRow, 0, len(rowsRaw))
	for _, r := range rowsRaw {
		rm, ok := r.(map[string]any)
		if !ok {
			continue
		}
		id := paramString(rm, "id")
		rows = append(rows, treasure.StatusRow{ID: id, Scope: paramStringSlice(rm, "scope")})
	}
	return rows, paramString(params, "value")
}

func applyScopeAssertions(assertions []Assertion, actualIDs []string, res *ScenarioResult) {
	for _, a := range assertions {
		var v *Violation
		//exhaustive:ignore -- only scope-includes/scope-excludes assertions apply to scope_filter scenarios
		switch a.Type {
		case AssertScopeIncludes:
			v = checkScopeIncludes(actualIDs, a)
		case AssertScopeExcludes:
			v = checkScopeExcludes(actualIDs, a)
		}
		if v != nil {
			res.Violations = append(res.Violations, *v)
		}
	}
}

func checkScopeIncludes(actualIDs []string, a Assertion) *Violation {
	if containsString(actualIDs, a.Value) {
		return nil
	}
	return &Violation{
		AssertionType: AssertScopeIncludes,
		Message:       "expected id missing from filtered result",
		Expected:      a.Value,
		Actual:        fmt.Sprintf("%v", actualIDs),
	}
}

func checkScopeExcludes(actualIDs []string, a Assertion) *Violation {
	if !containsString(actualIDs, a.Value) {
		return nil
	}
	return &Violation{
		AssertionType: AssertScopeExcludes,
		Message:       "unexpected id present in filtered result",
		Expected:      "absent: " + a.Value,
		Actual:        fmt.Sprintf("%v", actualIDs),
	}
}

func compareIDSets(res *ScenarioResult, expected, actual []string) {
	if len(expected) != len(actual) {
		res.Violations = append(res.Violations, Violation{
			Message:  "id set length mismatch",
			Expected: fmt.Sprintf("%v", expected),
			Actual:   fmt.Sprintf("%v", actual),
		})
		return
	}
	for _, id := range expected {
		if !containsString(actual, id) {
			res.Violations = append(res.Violations, Violation{
				Message:  "expected id missing",
				Expected: id,
				Actual:   fmt.Sprintf("%v", actual),
			})
		}
	}
}

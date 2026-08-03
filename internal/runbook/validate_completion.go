package runbook

import "fmt"

// Result is ValidateCompletion's verdict for a whole Runbook. Violations
// name mandatory checks that block completion; Exceptions name mandatory
// checks that passed only via a justification, recorded but non-blocking.
type Result struct {
	Passed     bool
	Violations []string
	Exceptions []string
}

// ValidateCompletion checks whether a Runbook's mandatory checks are all
// satisfied — either directly (satisfied[id] == true) or via an explicit
// entry in justifications. A mandatory check that is neither satisfied nor
// justified blocks completion; recommended/conditional/informational
// checks never block, per design.md § Completion Validation. Checks not
// present in satisfied are treated as unsatisfied, same as an explicit
// false.
func ValidateCompletion(rb Runbook, satisfied map[string]bool, justifications map[string]string) Result {
	result := Result{Passed: true}
	for _, check := range rb.Checks {
		if check.Level != LevelMandatory {
			continue
		}
		ok, excepted := evaluateSatisfaction(satisfied[check.ID], justifications[check.ID])
		if !ok {
			result.Passed = false
			result.Violations = append(result.Violations, fmt.Sprintf("mandatory check %q is not satisfied and has no justification", check.ID))
			continue
		}
		if excepted {
			result.Exceptions = append(result.Exceptions, fmt.Sprintf("mandatory check %q satisfied via exception: %s", check.ID, justifications[check.ID]))
		}
	}
	return result
}

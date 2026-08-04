package eval

import "time"

// ScenarioResult captures the outcome of running a single scenario.
type ScenarioResult struct {
	ScenarioID string
	Passed     bool
	Violations []Violation
	Duration   time.Duration
}

// Violation records one failed check within a scenario run.
type Violation struct {
	AssertionType AssertionType
	Message       string
	Expected      string
	Actual        string
}

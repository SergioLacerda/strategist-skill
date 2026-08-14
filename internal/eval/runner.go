package eval

import "time"

// SuiteResult captures one deterministic eval suite run.
type SuiteResult struct {
	SuiteID    string
	StartedAt  time.Time
	FinishedAt time.Time
	Duration   time.Duration
	Results    []ScenarioResult
	Passed     bool
}

// RunScenarios executes scenarios sequentially and aggregates their results.
func RunScenarios(suiteID string, scenarios []Scenario) SuiteResult {
	start := time.Now().UTC()
	results := make([]ScenarioResult, 0, len(scenarios))
	passed := true
	for _, s := range scenarios {
		res := RunScenario(s)
		results = append(results, res)
		if !res.Passed {
			passed = false
		}
	}
	finished := time.Now().UTC()
	return SuiteResult{
		SuiteID:    suiteID,
		StartedAt:  start,
		FinishedAt: finished,
		Duration:   finished.Sub(start),
		Results:    results,
		Passed:     passed,
	}
}

package eval

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// ReportFormat selects the machine-readable encoding WriteReport produces.
type ReportFormat string

// Supported report formats.
const (
	ReportJSON  ReportFormat = "json"
	ReportJUnit ReportFormat = "junit"
)

// WriteReport writes a deterministic machine-readable report for a suite run.
func WriteReport(w io.Writer, suite SuiteResult, format ReportFormat) error {
	switch format {
	case ReportJSON:
		return writeJSONReport(w, suite)
	case ReportJUnit:
		return writeJUnitReport(w, suite)
	default:
		return fmt.Errorf("eval report: unknown format %q (want json or junit)", format)
	}
}

type jsonSuiteReport struct {
	SuiteID    string               `json:"suite_id"`
	Passed     bool                 `json:"passed"`
	StartedAt  string               `json:"started_at"`
	FinishedAt string               `json:"finished_at"`
	DurationMS int64                `json:"duration_ms"`
	Counts     jsonReportCounts     `json:"counts"`
	Scenarios  []jsonScenarioReport `json:"scenarios"`
}

type jsonReportCounts struct {
	Total      int `json:"total"`
	Passed     int `json:"passed"`
	Failed     int `json:"failed"`
	Violations int `json:"violations"`
}

type jsonScenarioReport struct {
	ID         string                `json:"id"`
	Passed     bool                  `json:"passed"`
	DurationMS int64                 `json:"duration_ms"`
	Violations []jsonViolationReport `json:"violations,omitempty"`
}

type jsonViolationReport struct {
	AssertionType string `json:"assertion_type,omitempty"`
	Message       string `json:"message"`
	Expected      string `json:"expected,omitempty"`
	Actual        string `json:"actual,omitempty"`
}

func writeJSONReport(w io.Writer, suite SuiteResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(toJSONSuiteReport(suite)); err != nil {
		return fmt.Errorf("eval report: encode json: %w", err)
	}
	return nil
}

func toJSONSuiteReport(suite SuiteResult) jsonSuiteReport {
	out := jsonSuiteReport{
		SuiteID:    suite.SuiteID,
		Passed:     suite.Passed,
		StartedAt:  formatReportTime(suite.StartedAt),
		FinishedAt: formatReportTime(suite.FinishedAt),
		DurationMS: suite.Duration.Milliseconds(),
		Scenarios:  make([]jsonScenarioReport, 0, len(suite.Results)),
	}
	out.Counts.Total = len(suite.Results)
	for _, res := range suite.Results {
		scenario := jsonScenarioReport{
			ID:         res.ScenarioID,
			Passed:     res.Passed,
			DurationMS: res.Duration.Milliseconds(),
			Violations: make([]jsonViolationReport, 0, len(res.Violations)),
		}
		if res.Passed {
			out.Counts.Passed++
		} else {
			out.Counts.Failed++
		}
		out.Counts.Violations += len(res.Violations)
		for _, v := range res.Violations {
			scenario.Violations = append(scenario.Violations, jsonViolationReport{
				AssertionType: string(v.AssertionType),
				Message:       v.Message,
				Expected:      v.Expected,
				Actual:        v.Actual,
			})
		}
		out.Scenarios = append(out.Scenarios, scenario)
	}
	return out
}

// formatReportTime is shared by both report formats — see reporter_junit.go.
func formatReportTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

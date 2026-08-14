package eval

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"testing"
	"time"
)

func TestRunScenariosAggregatesResults(t *testing.T) {
	suite := RunScenarios("sample", []Scenario{
		{ID: "pass", Input: Input{Target: TargetStateMachine, Params: map[string]any{"start": "APPROVAL_GATE", "events": []any{"gate_approved"}}}, Expected: Expected{State: "EXECUTION"}},
		{ID: "fail", Input: Input{Target: TargetStateMachine, Params: map[string]any{"start": "APPROVAL_GATE", "events": []any{"gate_denied"}}}, Expected: Expected{State: "EXECUTION"}},
	})

	if suite.Passed {
		t.Fatalf("expected suite to fail when one scenario fails")
	}
	if len(suite.Results) != 2 {
		t.Fatalf("expected two scenario results, got %d", len(suite.Results))
	}
	if suite.Results[0].ScenarioID != "pass" || !suite.Results[0].Passed {
		t.Fatalf("first scenario did not pass as expected: %+v", suite.Results[0])
	}
	if suite.Results[1].ScenarioID != "fail" || suite.Results[1].Passed {
		t.Fatalf("second scenario did not fail as expected: %+v", suite.Results[1])
	}
}

func TestWriteReportJSON(t *testing.T) {
	suite := fixtureSuiteResult()
	var buf bytes.Buffer
	if err := WriteReport(&buf, suite, ReportJSON); err != nil {
		t.Fatalf("WriteReport JSON: %v", err)
	}

	var got struct {
		SuiteID string `json:"suite_id"`
		Passed  bool   `json:"passed"`
		Counts  struct {
			Total      int `json:"total"`
			Passed     int `json:"passed"`
			Failed     int `json:"failed"`
			Violations int `json:"violations"`
		} `json:"counts"`
		Scenarios []struct {
			ID         string `json:"id"`
			Passed     bool   `json:"passed"`
			Violations []struct {
				Message  string `json:"message"`
				Expected string `json:"expected"`
				Actual   string `json:"actual"`
			} `json:"violations"`
		} `json:"scenarios"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("report is not valid JSON: %v\n%s", err, buf.String())
	}
	if got.SuiteID != "eval/contracts" || got.Passed {
		t.Fatalf("unexpected suite fields: %+v", got)
	}
	if got.Counts.Total != 2 || got.Counts.Passed != 1 || got.Counts.Failed != 1 || got.Counts.Violations != 1 {
		t.Fatalf("unexpected counts: %+v", got.Counts)
	}
	if got.Scenarios[1].Violations[0].Expected != "EXECUTION" || got.Scenarios[1].Violations[0].Actual != "DONE_ANALYSIS" {
		t.Fatalf("unexpected violation: %+v", got.Scenarios[1].Violations[0])
	}
}

func TestWriteReportJUnit(t *testing.T) {
	suite := fixtureSuiteResult()
	var buf bytes.Buffer
	if err := WriteReport(&buf, suite, ReportJUnit); err != nil {
		t.Fatalf("WriteReport JUnit: %v", err)
	}

	var got junitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("report is not valid XML: %v\n%s", err, buf.String())
	}
	if got.Name != "eval/contracts" || got.Tests != 2 || got.Failures != 1 {
		t.Fatalf("unexpected testsuites attrs: %+v", got)
	}
	if len(got.Suites) != 1 || len(got.Suites[0].Cases) != 2 {
		t.Fatalf("unexpected testcase shape: %+v", got.Suites)
	}
	if got.Suites[0].Cases[1].Failures[0].Message != "final state mismatch" {
		t.Fatalf("unexpected failure: %+v", got.Suites[0].Cases[1].Failures)
	}
}

func TestWriteReportRejectsUnknownFormat(t *testing.T) {
	err := WriteReport(&bytes.Buffer{}, SuiteResult{}, ReportFormat("tap"))
	if err == nil {
		t.Fatalf("expected unknown report format error")
	}
}

func fixtureSuiteResult() SuiteResult {
	start := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	return SuiteResult{
		SuiteID:    "eval/contracts",
		StartedAt:  start,
		FinishedAt: start.Add(1500 * time.Millisecond),
		Duration:   1500 * time.Millisecond,
		Passed:     false,
		Results: []ScenarioResult{
			{ScenarioID: "pass", Passed: true, Duration: 500 * time.Millisecond},
			{ScenarioID: "fail", Passed: false, Duration: time.Second, Violations: []Violation{{
				AssertionType: AssertEqualState,
				Message:       "final state mismatch",
				Expected:      "EXECUTION",
				Actual:        "DONE_ANALYSIS",
			}}},
		},
	}
}

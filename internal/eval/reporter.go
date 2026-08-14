package eval

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"time"
)

type ReportFormat string

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
	return enc.Encode(toJSONSuiteReport(suite))
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

type junitTestSuites struct {
	XMLName  xml.Name         `xml:"testsuites"`
	Name     string           `xml:"name,attr"`
	Tests    int              `xml:"tests,attr"`
	Failures int              `xml:"failures,attr"`
	Time     string           `xml:"time,attr"`
	Suites   []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Time      string          `xml:"time,attr"`
	Timestamp string          `xml:"timestamp,attr,omitempty"`
	Cases     []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	Classname string         `xml:"classname,attr"`
	Name      string         `xml:"name,attr"`
	Time      string         `xml:"time,attr"`
	Failures  []junitFailure `xml:"failure,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr,omitempty"`
	Text    string `xml:",chardata"`
}

func writeJUnitReport(w io.Writer, suite SuiteResult) error {
	report := toJUnitReport(suite)
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(report); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func toJUnitReport(suite SuiteResult) junitTestSuites {
	failures := 0
	cases := make([]junitTestCase, 0, len(suite.Results))
	for _, res := range suite.Results {
		tc := junitTestCase{
			Classname: suite.SuiteID,
			Name:      res.ScenarioID,
			Time:      secondsString(res.Duration),
			Failures:  make([]junitFailure, 0, len(res.Violations)),
		}
		for _, v := range res.Violations {
			failures++
			tc.Failures = append(tc.Failures, junitFailure{
				Message: v.Message,
				Type:    string(v.AssertionType),
				Text:    formatViolation(v),
			})
		}
		cases = append(cases, tc)
	}
	return junitTestSuites{
		Name:     suite.SuiteID,
		Tests:    len(suite.Results),
		Failures: failures,
		Time:     secondsString(suite.Duration),
		Suites: []junitTestSuite{{
			Name:      suite.SuiteID,
			Tests:     len(suite.Results),
			Failures:  failures,
			Time:      secondsString(suite.Duration),
			Timestamp: formatReportTime(suite.StartedAt),
			Cases:     cases,
		}},
	}
}

func formatViolation(v Violation) string {
	return fmt.Sprintf("expected: %s\nactual: %s", v.Expected, v.Actual)
}

func formatReportTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func secondsString(d time.Duration) string {
	return fmt.Sprintf("%.3f", d.Seconds())
}

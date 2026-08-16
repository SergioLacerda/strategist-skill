package eval

import (
	"encoding/xml"
	"fmt"
	"io"
	"time"
)

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
		return fmt.Errorf("eval report: write xml header: %w", err)
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(report); err != nil {
		return fmt.Errorf("eval report: encode junit xml: %w", err)
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return fmt.Errorf("eval report: write trailing newline: %w", err)
	}
	return nil
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

func secondsString(d time.Duration) string {
	return fmt.Sprintf("%.3f", d.Seconds())
}

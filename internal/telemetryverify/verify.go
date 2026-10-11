// Package telemetryverify checks the integrity of a mission's telemetry
// evidence: the lifecycle ledger, the usage ledger and, when supplied, the
// exported event stream. It is read-only and advisory: it reports findings and
// never repairs, rewrites or blocks a mission. An Audit Addon can consume the
// same report without Strategist depending on it.
package telemetryverify

import (
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// Severity ranks a finding.
type Severity string

// Severities. Only an error means the evidence contradicts itself.
const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Finding is one integrity observation. Ref names the record it concerns
// (marker id, record id or event name); it never carries token content.
type Finding struct {
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Ref      string   `json:"ref,omitempty"`
}

// Input is everything the verifier reads. Malformed counts come from reading
// the raw files: the ledger readers skip such lines silently.
type Input struct {
	Status             domain.MissionEngineStatus
	Lifecycle          []telemetry.MissionLifecycleRecord
	Usage              []telemetry.MissionUsageRecord
	KnownInvocations   []string
	LifecycleMalformed int
	UsageMalformed     int
	Events             []telemetry.Event
	EventsProvided     bool
	EventsMalformed    int
}

// Checked counts what was inspected.
type Checked struct {
	LifecycleRecords int `json:"lifecycle_records"`
	UsageRecords     int `json:"usage_records"`
	Events           int `json:"events"`
}

// Report is the verifier output.
type Report struct {
	MissionID string         `json:"mission_id"`
	OK        bool           `json:"ok"`
	Checked   Checked        `json:"checked"`
	Counts    map[string]int `json:"counts"`
	Findings  []Finding      `json:"findings"`
}

// Verify runs every check. OK is true when there is no error-level finding.
func Verify(in Input) Report {
	var findings []Finding
	findings = append(findings, malformedFindings(in)...)
	findings = append(findings, lifecycleFindings(in)...)
	findings = append(findings, usageFindings(in)...)
	if in.EventsProvided {
		findings = append(findings, eventFindings(in)...)
	}
	sort.SliceStable(findings, func(i, j int) bool { return rank(findings[i].Severity) > rank(findings[j].Severity) })
	counts := map[string]int{string(SeverityError): 0, string(SeverityWarning): 0, string(SeverityInfo): 0}
	for _, f := range findings {
		counts[string(f.Severity)]++
	}
	return Report{
		MissionID: in.Status.MissionID, OK: counts[string(SeverityError)] == 0,
		Checked: Checked{LifecycleRecords: len(in.Lifecycle), UsageRecords: len(in.Usage), Events: len(in.Events)},
		Counts:  counts, Findings: nonNil(findings),
	}
}

func rank(s Severity) int {
	switch s {
	case SeverityError:
		return 2
	case SeverityWarning:
		return 1
	case SeverityInfo:
		return 0
	}
	return 0
}

func nonNil(f []Finding) []Finding {
	if f == nil {
		return []Finding{}
	}
	return f
}

func malformedFindings(in Input) []Finding {
	var out []Finding
	for name, n := range map[string]int{"lifecycle ledger": in.LifecycleMalformed, "usage ledger": in.UsageMalformed, "event stream": in.EventsMalformed} {
		if n > 0 {
			out = append(out, Finding{
				Code: "unreadable_lines", Severity: SeverityWarning, Ref: name,
				Message: name + " has lines that are not valid records; readers skip them silently",
			})
		}
	}
	return out
}

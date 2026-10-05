package calibration

import (
	"regexp"
	"sort"
	"strings"
)

// LocalCheck is a deterministic check over the projected fields of a boundary: it
// flags a defect with no provider involved. The DOJO measures it like the judge
// (how many defects it catches, how many good packages it wrongly flags), so the
// protection of an operator the judge cannot see can be delegated to it and proven.
type LocalCheck struct {
	Name     string
	Boundary string
	Flags    func(Fields) bool
}

var (
	anyTask      = regexp.MustCompile(`(?m)^\s*- \[[ xX]\] \d+(?:\.\d+)* `)
	docTask      = regexp.MustCompile(`(?m)^\s*- \[[ xX]\] \d+(?:\.\d+)* \[analysis_artifact\] .*$`)
	codeMarker   = regexp.MustCompile(`(?i)\binternal/|\bcmd/|\.go\b|go test`)
	untypedTasks = regexp.MustCompile(`(?m)^\s*- \[[ xX]\] \d+(?:\.\d+)* [^\[]`)
)

// LocalChecks returns the reference local checks of every boundary, ordered by name.
func LocalChecks() []LocalCheck {
	checks := []LocalCheck{
		{"scope_missing", BoundaryPackage, func(f Fields) bool { return !hasField(f, "approved_scope") }},
		{"acceptance_missing", BoundaryPackage, func(f Fields) bool { return !hasField(f, "acceptance_checks") }},
		{"plan_missing", BoundaryPackage, func(f Fields) bool { return !hasField(f, "implementation_plan") }},
		{"task_untyped", BoundaryPackage, func(f Fields) bool { return untypedTasks.MatchString(f["implementation_plan"]) }},
		{"task_unverified", BoundaryPackage, taskUnverified},
		{"task_kind_mismatch", BoundaryPackage, kindMismatch},
		{"objective_missing", BoundaryRanger, func(f Fields) bool { return !hasField(f, "objective") }},
		{"facts_unsourced", BoundaryRanger, func(f Fields) bool { return hasField(f, "known_facts") && !pathToken.MatchString(f["known_facts"]) }},
		{"uncertainties_missing", BoundaryRanger, func(f Fields) bool { return !hasField(f, "uncertainties") }},
		{"focus_missing", BoundaryRanger, func(f Fields) bool { return !hasField(f, "recommended_refinement_focus") }},
	}
	sort.Slice(checks, func(i, j int) bool { return checks[i].Name < checks[j].Name })
	return checks
}

// taskUnverified flags a plan with a task that states no verification.
func taskUnverified(f Fields) bool {
	for _, line := range lines(f["implementation_plan"]) {
		if anyTask.MatchString(line) && !strings.Contains(strings.ToLower(line), "verify") {
			return true
		}
	}
	return false
}

// kindMismatch flags a task declared as documentation work that names code to
// build or run: its declared kind and its action disagree. It is deliberately
// plain (path and command markers); the DOJO measures how many good packages it
// wrongly flags before anyone relies on it.
func kindMismatch(f Fields) bool {
	for _, line := range docTask.FindAllString(f["implementation_plan"], -1) {
		if codeMarker.MatchString(line) {
			return true
		}
	}
	return false
}

// LocalFlags returns the names of the local checks of a boundary that flag the
// fields, sorted. An empty result means none flagged, not that none ran.
func LocalFlags(boundary string, fields Fields) []string {
	var names []string
	for _, check := range LocalChecks() {
		if check.Boundary == boundary && check.Flags(fields) {
			names = append(names, check.Name)
		}
	}
	return names
}

package calibration

import "strings"

// semanticPackageOperators are the stage 2 operators: their label is known by
// construction but the defect is semantic, so an item counts in the golden
// metrics only after it was audited (the DOJO audit flow).
func semanticPackageOperators() []Operator {
	stage2 := func(name string, band Band, criteria []string, applies func(Fields) bool, apply func(Fields, Context) Fields) Operator {
		return Operator{Name: name, Stage: 2, Boundary: BoundaryPackage, Band: band, Expect: ExpectReject, Criteria: criteria, Applies: applies, Apply: apply}
	}
	return []Operator{
		stage2("swap_plan", BandMajor, nil, func(f Fields) bool { return hasField(f, "implementation_plan") }, swapPlan),
		stage2("false_closure", BandMajor, []string{"implementation_not_materialized"}, func(f Fields) bool { return checkedTask.MatchString(f["implementation_plan"]) }, falseClosure),
		stage2("scope_drift", BandMinor, []string{"scope_bounded"}, func(f Fields) bool { return hasField(f, "approved_scope") && hasField(f, "implementation_plan") }, scopeDrift),
		stage2("unsupported_provenance_claim", BandMinor, nil, func(f Fields) bool { return hasField(f, "implementation_plan") }, provenanceClaim),
		stage2("ambiguous_targets", BandMinor, []string{"acceptance_checkable"}, func(f Fields) bool {
			return pathToken.MatchString(f["acceptance_checks"]) || pathToken.MatchString(f["implementation_plan"])
		}, ambiguousTargets),
	}
}

func swapPlan(f Fields, ctx Context) Fields {
	if !hasField(ctx.Other, "implementation_plan") {
		return f
	}
	return with(f, "implementation_plan", ctx.Other["implementation_plan"])
}

func falseClosure(f Fields, _ Context) Fields {
	closed := checkedTask.ReplaceAllString(f["implementation_plan"], "- [x]")
	return with(f, "implementation_plan", closed+"\n\nAll tasks are implemented and verified.")
}

func scopeDrift(f Fields, _ Context) Fields {
	return with(f, "implementation_plan", f["implementation_plan"]+"\n- [ ] 9.9 [implementation_handoff] Rewrite the billing module under `internal/billing/**`; verify the billing tests pass.")
}

func provenanceClaim(f Fields, _ Context) Fields {
	return with(f, "implementation_plan", f["implementation_plan"]+"\n\nThis plan was independently verified by three reviewers.")
}

func ambiguousTargets(f Fields, _ Context) Fields {
	vague := func(text string) string { return pathToken.ReplaceAllString(text, "the relevant files") }
	out := with(f, "implementation_plan", vague(f["implementation_plan"]))
	if hasField(f, "acceptance_checks") {
		out["acceptance_checks"] = strings.TrimSpace(vague(f["acceptance_checks"]))
	}
	return out
}

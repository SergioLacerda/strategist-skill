package calibration

import (
	"regexp"
	"strings"
)

var gradedLevels = []float64{0.25, 0.5, 1}

func planHas(re *regexp.Regexp) func(Fields) bool {
	return func(f Fields) bool {
		return hasField(f, "implementation_plan") && re.MatchString(f["implementation_plan"])
	}
}

func tasksPresent(f Fields) bool {
	return hasField(f, "implementation_plan") && len(taskIndexes(f["implementation_plan"])) > 0
}

func verifiedTasks(f Fields) bool {
	for _, i := range taskIndexes(f["implementation_plan"]) {
		if verifyTail.MatchString(lines(f["implementation_plan"])[i]) {
			return true
		}
	}
	return false
}

func packageOperators() []Operator {
	pkg := func(name string, band Band, expect Expect, criteria []string, applies func(Fields) bool, apply func(Fields, Context) Fields) Operator {
		return Operator{Name: name, Stage: 1, Boundary: BoundaryPackage, Band: band, Expect: expect, Criteria: criteria, Applies: applies, Apply: apply}
	}
	graded := func(op Operator) Operator { op.Levels = gradedLevels; return op }
	return []Operator{
		pkg("drop_approved_scope", BandMajor, ExpectReject, []string{"scope_bounded"}, func(f Fields) bool { return hasField(f, "approved_scope") },
			func(f Fields, _ Context) Fields { return with(f, "approved_scope", "") }),
		pkg("contradict_scope", BandMajor, ExpectReject, []string{"scope_bounded"}, func(f Fields) bool { return hasField(f, "approved_scope") },
			func(f Fields, _ Context) Fields {
				return with(f, "approved_scope", f["approved_scope"]+"\n- allowed: every path in the repository\n- forbidden: every path in the repository")
			}),
		graded(pkg("strip_task_types", BandMajor, ExpectReject, []string{"tasks_classified"}, tasksPresent, stripTaskTypes)),
		pkg("strip_all_task_types", BandMajor, ExpectReject, []string{"tasks_classified"}, tasksPresent, func(f Fields, ctx Context) Fields {
			ctx.Level = 1
			return stripTaskTypes(f, ctx)
		}),
		graded(pkg("drop_task_verification", BandMajor, ExpectReject, []string{"validation_stated"}, verifiedTasks, dropVerification)),
		pkg("relabel_implementation_as_documentation", BandMajor, ExpectReject, []string{"implementation_not_materialized"},
			planHas(regexp.MustCompile(`\[implementation_handoff\]`)), relabelImplementation),
		pkg("empty_acceptance_checks", BandMajor, ExpectReject, []string{"acceptance_checkable"}, func(f Fields) bool { return hasField(f, "acceptance_checks") },
			func(f Fields, _ Context) Fields { return with(f, "acceptance_checks", "") }),
		pkg("truncate_plan", BandMinor, ExpectReject, []string{"validation_stated"}, func(f Fields) bool { return len(f["implementation_plan"]) > 200 },
			func(f Fields, _ Context) Fields {
				plan := f["implementation_plan"]
				return with(f, "implementation_plan", plan[:len(plan)*2/5])
			}),
		pkg("weaken_must_to_should", BandMinor, ExpectDrop, nil, planHas(mustWord), func(f Fields, _ Context) Fields {
			return with(f, "implementation_plan", mustWord.ReplaceAllString(f["implementation_plan"], "SHOULD"))
		}),
	}
}

func stripTaskTypes(f Fields, ctx Context) Fields {
	return with(f, "implementation_plan", rewriteTasks(f["implementation_plan"], ctx, func(line string) string {
		return taskLine.ReplaceAllString(line, "$1$3")
	}))
}

func dropVerification(f Fields, ctx Context) Fields {
	return with(f, "implementation_plan", rewriteTasks(f["implementation_plan"], ctx, func(line string) string {
		return strings.TrimRight(verifyTail.ReplaceAllString(line, ""), " ") + "."
	}))
}

func relabelImplementation(f Fields, ctx Context) Fields {
	return with(f, "implementation_plan", rewriteTasks(f["implementation_plan"], ctx, func(line string) string {
		return strings.Replace(line, "[implementation_handoff]", "[analysis_artifact]", 1)
	}))
}

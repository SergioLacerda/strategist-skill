package calibration

import (
	"fmt"
	"strings"
)

const samplePlan = `- [ ] 1.1 [analysis_artifact] Record the answers in design.md; verify D1 is updated.
- [ ] 1.2 [implementation_handoff] Refresh the README and relocate the stray package; verify dojo list and the README agree.
- [ ] 2.1 [implementation_handoff] Build the manifest builder; verify tests cover inclusion and determinism.
- [ ] 2.2 [implementation_handoff] Build the catalog; verify a golden test per operator. The agent MUST NOT relabel a case.
- [ ] 3.1 [implementation_handoff] Build the evaluator; verify it against a scripted judge.
- [ ] 3.2 [analysis_artifact] Report the first calibration; verify it states measured values only.`

func packageFields(variant int) Fields {
	return Fields{
		"implementation_plan": strings.ReplaceAll(samplePlan, "README", fmt.Sprintf("README-%d", variant)),
		"approved_scope":      "- allowed: docs/**, internal/dojo/**\n- forbidden: internal/domain/**",
		"acceptance_checks":   "- `go test ./...` passes\n- `strategist dojo list` lists no stray directory",
	}
}

func rangerFields() Fields {
	return Fields{
		"objective":                    "Decide how the DOJO should calibrate the provider pre-check.",
		"known_facts":                  "- The README lists scenarios that do not exist (`.analysis/dojo/README.md`).",
		"uncertainties":                "- Whether real package fields may be sent to the provider.",
		"recommended_refinement_focus": "Specify the corpus, the defect catalog and the calibration record.",
	}
}

// ruleJudge mimics the criteria catalog on the text: a package scores high only
// when scope, task types, verification and acceptance checks are all present.
func ruleJudge(_ string, f Fields) Verdict {
	score := 0.95
	plan := f["implementation_plan"]
	if strings.TrimSpace(f["approved_scope"]) == "" || strings.Contains(f["approved_scope"], "every path") {
		score -= 0.5
	}
	tasks := strings.Count(plan, "- [")
	typed := len(taskIndexes(plan))
	score -= 0.4 * float64(tasks-typed) / float64(max(tasks, 1))
	if strings.Count(strings.ToLower(plan), "verify") < tasks {
		score -= 0.3
	}
	if strings.TrimSpace(f["acceptance_checks"]) == "" {
		score -= 0.3
	}
	if strings.Count(plan, "[implementation_handoff]") < 4 {
		score -= 0.5
	}
	if !strings.HasSuffix(strings.TrimSpace(plan), ".") {
		score -= 0.5
	}
	if !strings.Contains(plan, "MUST") {
		score -= 0.1
	}
	return ScoredVerdict(max(score, 0.02))
}

package calibration

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const codePlan = `- [ ] 1.1 [analysis_artifact] Record the answers in docs/x/design-%[1]d.md; verify the file lists each answer.
- [ ] 2.1 [implementation_handoff] Build the package under internal/x%[1]d; verify its tests cover the empty case.
- [ ] 2.2 [implementation_handoff] Wire it in cmd/strategist/x%[1]d.go. The wiring MUST NOT change a flag; verify ` + "`go test ./cmd/strategist/...`" + ` exits 0.
- [ ] 3.1 [analysis_artifact] Report what x%[1]d measured in docs/x/README.md; verify the report states measured values only.
`

func codeFields(variant int) Fields {
	return Fields{
		"implementation_plan": fmt.Sprintf(codePlan, variant),
		"approved_scope":      "- allowed: docs/x/**, internal/x" + fmt.Sprint(variant) + "/**\n- forbidden: internal/domain/**",
		"acceptance_checks":   "- `go test ./...` passes",
	}
}

func codeSource(variant int) Source {
	f := codeFields(variant)
	return Source{Name: "pkg", Digest: Digest(f), Boundary: BoundaryPackage, Fields: f, Golden: true, HeldOut: HeldOut(Digest(f), "t", 25)}
}

func TestAnIntactPackageIsFlaggedByNoLocalCheck(t *testing.T) {
	assert.Empty(t, LocalFlags(BoundaryPackage, codeFields(1)))
}

func TestEachLocalCheckCatchesItsOwnDefect(t *testing.T) {
	flagged := func(operator string) []string {
		for _, c := range Generate(codeSource(1), Options{Seed: 1, Operators: []string{operator}}) {
			if c.Operator == operator && c.Level >= 1 {
				return LocalFlags(BoundaryPackage, c.Fields)
			}
		}
		return nil
	}
	assert.Contains(t, flagged("drop_approved_scope"), "scope_missing")
	assert.Contains(t, flagged("empty_acceptance_checks"), "acceptance_missing")
	assert.Contains(t, flagged("strip_all_task_types"), "task_untyped")
	assert.Contains(t, flagged("drop_task_verification"), "task_unverified")
	assert.Contains(t, flagged("relabel_implementation_as_documentation"), "task_kind_mismatch")
	assert.Empty(t, flagged("weaken_must_to_should"), "nothing local sees a change of modality")
	assert.Empty(t, flagged("collapse_spaces"), "a neutral change is not flagged")
}

func TestARangerArtifactHasItsOwnLocalChecks(t *testing.T) {
	assert.Empty(t, LocalFlags(BoundaryRanger, rangerFields()))
	f := rangerFields()
	delete(f, "objective")
	delete(f, "uncertainties")
	assert.Equal(t, []string{"objective_missing", "uncertainties_missing"}, LocalFlags(BoundaryRanger, f))
}

func localObservations(t *testing.T, packages int) []Observation {
	t.Helper()
	var cases []Case
	for i := 0; i < packages; i++ {
		cases = append(cases, Generate(codeSource(i), Options{Seed: 1})...)
	}
	var out []Observation
	_, err := Run(context.Background(), cases, FuncJudge(codeJudge), RunOptions{Repetitions: 2, Local: LocalFlags, Sink: func(o Observation) error { out = append(out, o); return nil }})
	require.NoError(t, err)
	return out
}

func TestObservationsCarryTheLocalFlagsOfTheirCaseAndNoContent(t *testing.T) {
	for _, o := range localObservations(t, 3) {
		assert.True(t, o.Local)
		if o.Operator == "" {
			assert.Empty(t, o.LocalFlags)
		}
		raw, err := o.Encode()
		require.NoError(t, err)
		assert.NotContains(t, string(raw), "internal/x")
	}
}

func TestThePolicyLevelCountsADefectCaughtByEitherProtection(t *testing.T) {
	cfg := DefaultConfig()
	report := Evaluate(localObservations(t, 60), cfg)
	require.NotNil(t, report.PolicyLevel)
	rows := map[string]CoverageRow{}
	for _, row := range report.PolicyLevel.Coverage {
		rows[row.Operator] = row
	}
	relabel := rows["relabel_implementation_as_documentation"]
	assert.Equal(t, relabel.Cases, relabel.Local.K, "the local check flags every relabelled package")
	assert.Equal(t, relabel.Cases, relabel.Policy.K)
	assert.Zero(t, report.PolicyLevel.GoldenFlagged.K)
	assert.Zero(t, report.PolicyLevel.FalseApproval.K)
}

func TestAnOperatorDelegatedToTheLocalChecksLeavesTheJudgeGatesButNeedsLocalCoverage(t *testing.T) {
	obs := localObservations(t, 60)
	blindToRelabel := make([]Observation, len(obs))
	copy(blindToRelabel, obs)
	for i := range blindToRelabel {
		if blindToRelabel[i].Operator == "relabel_implementation_as_documentation" {
			blindToRelabel[i].PAccept = 0.95 // the judge approves a relabelled package
		}
	}

	plain := Evaluate(blindToRelabel, DefaultConfig())
	assert.Equal(t, GateSeparation, firstFailing(plain.Gates), "left in the judge set, an unseen operator fails the judge gates")

	cfg := DefaultConfig()
	cfg.LocalScope = map[string]bool{"relabel_implementation_as_documentation": true}
	delegated := Evaluate(blindToRelabel, cfg)
	assert.Equal(t, []string{"relabel_implementation_as_documentation"}, delegated.LocalScope)
	last := delegated.Gates[len(delegated.Gates)-1]
	assert.Equal(t, GateLocalCoverage, last.Name)
	assert.Equal(t, GatePassed, last.Status, "%+v", last)
	assert.Equal(t, StatusRecommended, delegated.Status, "%v %v", delegated.Notes, delegated.Gates)
}

func TestADelegatedOperatorNothingLocalCatchesIsNotCalibrated(t *testing.T) {
	obs := localObservations(t, 60)
	for i := range obs {
		if obs[i].Operator == "weaken_must_to_should" {
			obs[i].PAccept = 0.95
		}
	}
	cfg := DefaultConfig()
	cfg.LocalScope = map[string]bool{"weaken_must_to_should": true}
	report := Evaluate(obs, cfg)
	assert.Equal(t, StatusNotFit, report.Status)
	assert.Equal(t, GateLocalCoverage, report.FailingGate)
}

func firstFailing(gates []Gate) string { return FirstOpenGate(gates) }

func TestWithoutLocalFlagsThereIsNoPolicyLevel(t *testing.T) {
	report := Evaluate(runAll(t, corpus(40), FuncJudge(ruleJudge), 2), DefaultConfig())
	assert.Nil(t, report.PolicyLevel)
}

// codeJudge is ruleJudge for the code plan, which has two implementation tasks.
func codeJudge(b string, f Fields) Verdict {
	if strings.Count(f["implementation_plan"], "[implementation_handoff]") < 2 {
		return ScoredVerdict(0.15)
	}
	patched := Fields{}
	for k, v := range f {
		patched[k] = v
	}
	patched["implementation_plan"] = strings.ReplaceAll(f["implementation_plan"], "[implementation_handoff]", "[implementation_handoff][implementation_handoff]")
	return ruleJudge(b, patched)
}

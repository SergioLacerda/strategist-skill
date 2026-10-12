package calibration

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func source(variant int) Source {
	fields := packageFields(variant)
	return Source{Name: "pkg", Digest: Digest(fields), Boundary: BoundaryPackage, Fields: fields, Golden: true}
}

func TestEveryPackageOperatorChangesTheFieldsAndStillProjects(t *testing.T) {
	src := source(1)
	cases := Generate(src, Options{Seed: 7, MaxStage: 2, Others: []Source{source(2)}})
	seen := map[string]bool{}
	for _, c := range cases[1:] {
		seen[c.Operator] = true
		assert.NotEqual(t, src.Fields, c.Fields, c.ID)
		assert.True(t, projects(c.Fields), c.ID)
		assert.Equal(t, BaseCase(src).ID, c.ParentID)
	}
	for _, name := range []string{"drop_approved_scope", "contradict_scope", "strip_task_types", "drop_task_verification",
		"relabel_implementation_as_documentation", "empty_acceptance_checks", "truncate_plan", "weaken_must_to_should",
		"swap_plan", "false_closure", "scope_drift", "unsupported_provenance_claim", "ambiguous_targets"} {
		assert.True(t, seen[name], name)
	}
}

func TestGenerateIsDeterministic(t *testing.T) {
	a := Generate(source(1), Options{Seed: 3, MaxStage: 2, Others: []Source{source(2)}})
	b := Generate(source(1), Options{Seed: 3, MaxStage: 2, Others: []Source{source(2)}})
	require.Len(t, b, len(a))
	for i := range a {
		assert.Equal(t, a[i].ID, b[i].ID)
		assert.Equal(t, a[i].Fields, b[i].Fields)
	}
}

func TestOperatorsHonorTheirPreconditions(t *testing.T) {
	bare := Source{Name: "bare", Digest: "sha256:bare", Boundary: BoundaryPackage, Fields: Fields{"implementation_plan": "no tasks here"}}
	for _, c := range Generate(bare, Options{Seed: 1, MaxStage: 2}) {
		assert.NotContains(t, []string{"drop_approved_scope", "strip_task_types", "empty_acceptance_checks", "relabel_implementation_as_documentation"}, c.Operator)
	}
}

func TestStage2OperatorsNeedTheirStage(t *testing.T) {
	for _, c := range Generate(source(1), Options{Seed: 1, Others: []Source{source(2)}}) {
		assert.Less(t, c.Stage, 2, c.ID)
	}
	var stage2 int
	for _, c := range Generate(source(1), Options{Seed: 1, MaxStage: 2, Others: []Source{source(2)}}) {
		if c.Stage == 2 {
			stage2++
			assert.False(t, c.Counted(), "an unaudited stage 2 case does not count: %s", c.ID)
		}
	}
	assert.Positive(t, stage2)
}

func TestGradedLevelsAreNestedAndExpectOnlyTheFullDefectRejected(t *testing.T) {
	cases := Generate(source(1), Options{Seed: 5, Operators: []string{"strip_task_types"}})
	require.Len(t, cases, 4) // base plus three levels
	assert.Equal(t, ExpectDrop, cases[1].Expect)
	assert.Equal(t, ExpectDrop, cases[2].Expect)
	assert.Equal(t, ExpectReject, cases[3].Expect)
	assert.Less(t, len(taskIndexes(cases[1].Fields["implementation_plan"])), len(taskIndexes(source(1).Fields["implementation_plan"])))
	assert.LessOrEqual(t, len(taskIndexes(cases[3].Fields["implementation_plan"])), len(taskIndexes(cases[2].Fields["implementation_plan"])))
	for _, c := range cases[1:] {
		assert.Equal(t, KindBoundary, c.Kind)
	}
}

func TestNeutralPerturbationsKeepTheMeaning(t *testing.T) {
	messy := Fields{"implementation_plan": "- [ ] 1.1 [analysis_artifact] Do  the thing;   verify it. \n\n\n\n- [ ] 1.2 [analysis_artifact] Next; verify it.", "approved_scope": "- allowed: docs/**"}
	src := Source{Name: "m", Digest: Digest(messy), Boundary: BoundaryPackage, Fields: messy, Golden: true}
	neutral := 0
	for _, c := range Generate(src, Options{Seed: 1}) {
		if c.Kind != KindNeutral {
			continue
		}
		neutral++
		assert.Equal(t, ExpectAccept, c.Expect)
		assert.Equal(t, strings.Fields(messy["implementation_plan"]), strings.Fields(c.Fields["implementation_plan"]))
	}
	assert.Equal(t, 3, neutral)
}

func TestRangerOperators(t *testing.T) {
	fields := rangerFields()
	src := Source{Name: "r", Digest: Digest(fields), Boundary: BoundaryRanger, Fields: fields, Golden: true}
	byOp := map[string]Case{}
	for _, c := range Generate(src, Options{Seed: 1}) {
		byOp[c.Operator] = c
	}
	assert.Equal(t, []string{"objective_clear"}, byOp["vague_objective"].Criteria)
	assert.NotContains(t, byOp["facts_without_sources"].Fields["known_facts"], ".md")
	assert.Empty(t, byOp["uncertainties_as_facts"].Fields["uncertainties"])
	assert.Contains(t, byOp["uncertainties_as_facts"].Fields["known_facts"], "Whether real package fields")
	assert.NotContains(t, byOp["empty_refinement_focus"].Fields, "recommended_refinement_focus")
}

func TestSwapPlanUsesAnotherSourceNeverItself(t *testing.T) {
	a, b := source(1), source(2)
	for _, c := range Generate(a, Options{Seed: 1, MaxStage: 2, Operators: []string{"swap_plan"}, Others: []Source{a, b}}) {
		if c.Operator == "swap_plan" {
			assert.Equal(t, b.Fields["implementation_plan"], c.Fields["implementation_plan"])
		}
	}
	assert.Len(t, Generate(a, Options{Seed: 1, MaxStage: 2, Operators: []string{"swap_plan"}, Others: []Source{a}}), 1, "no other source: the operator is skipped")
}

func TestCatalogVersionChangesWithTheCatalogAndIsStable(t *testing.T) {
	assert.True(t, strings.HasPrefix(CatalogVersion(), "catalog-"))
	_, ok := ByName("drop_approved_scope")
	assert.True(t, ok)
	_, ok = ByName("nope")
	assert.False(t, ok)
}

func TestNaturalNegativeIsPairedWithItsParent(t *testing.T) {
	base := BaseCase(source(1))
	natural := NaturalNegative(base, 1, packageFields(9))
	assert.Equal(t, base.ID, natural.ParentID)
	assert.Equal(t, KindNatural, natural.Kind)
	assert.Equal(t, ExpectReject, natural.Expect)
	assert.True(t, natural.Golden)
}

func TestHeldOutIsStableAndInheritedByChildren(t *testing.T) {
	first := HeldOut("sha256:aa", "s", 25)
	assert.Equal(t, first, HeldOut("sha256:aa", "s", 25))
	assert.False(t, HeldOut("sha256:aa", "s", 0))
	assert.True(t, HeldOut("x", "s", 100))
	src := source(3)
	src.HeldOut = true
	for _, c := range Generate(src, Options{Seed: 1}) {
		assert.True(t, c.HeldOut, c.ID)
	}
	held := 0
	for i := 0; i < 400; i++ {
		if HeldOut(Digest(packageFields(i)), "salt", 25) {
			held++
		}
	}
	assert.InDelta(t, 100, held, 30)
}

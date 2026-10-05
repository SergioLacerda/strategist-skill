package dojo_test

import (
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func baseTerm() dojo.ConsentTerm {
	return dojo.NewTerm(dojo.ConsentTerm{Case: "jev-handoff", Layer: "judge", Provider: "jev", Endpoint: "https://api.example/v1", DataClass: dojo.DataReal,
		Fields: "implementation_plan", Corpus: "sha256:c", Cases: 10, Calls: 30, Bytes: 4000, EstimatedTokens: 8500})
}

func TestAConsentTermIsDerivedFromItsPlan(t *testing.T) {
	term := baseTerm()
	assert.Len(t, term.ID, 12)
	assert.Equal(t, term.ID, baseTerm().ID)

	changed := term
	changed.Bytes++
	assert.NotEqual(t, term.ID, dojo.NewTerm(changed).ID, "any change of the plan is another term")
	other := term
	other.Endpoint = "https://elsewhere.example"
	assert.NotEqual(t, term.ID, dojo.NewTerm(other).ID)

	text := term.Text()
	for _, want := range []string{term.ID, "https://api.example/v1", dojo.DataReal, "calls: 30", "--consent", "nothing was sent now"} {
		assert.Contains(t, text, want)
	}
}

func TestAssentRecordsTheTermAndRefusesAChangedPlan(t *testing.T) {
	term := baseTerm()
	record, err := dojo.Assent(term, "", runTime)
	require.NoError(t, err)
	assert.False(t, record.Bound)
	assert.Equal(t, term.ID, record.Term.ID)

	record, err = dojo.Assent(term, term.ID, runTime)
	require.NoError(t, err)
	assert.True(t, record.Bound)

	_, err = dojo.Assent(term, "aaaaaaaaaaaa", runTime)
	require.ErrorContains(t, err, "the plan changed")
}

func TestSyntheticPackagesAreWellFormedAndContainNothingReal(t *testing.T) {
	dir := t.TempDir()
	names, err := dojo.WriteSyntheticPackages(dir, 3)
	require.NoError(t, err)
	require.Len(t, names, 3)
	for _, name := range names {
		for _, file := range []string{"analysis.md", "proposal.md", "design.md", "tasks.md"} {
			assert.FileExists(t, filepath.Join(dir, name, file))
		}
	}
	more, err := dojo.WriteSyntheticPackages(t.TempDir(), 1000)
	require.NoError(t, err)
	assert.Len(t, more, len(dojo.SyntheticTopics), "capped at the number of topics")
	few, err := dojo.WriteSyntheticPackages(t.TempDir(), 0)
	require.NoError(t, err)
	assert.Len(t, few, 1)
}

func TestSyntheticFixturesAreGoldenByConstructionOnlyUnderTheDojoDomain(t *testing.T) {
	w := newWorkspace(t)
	_, err := dojo.WriteSyntheticPackages(filepath.Join(w.base, "dojo", "pilot", "fixtures", "done"), 4)
	require.NoError(t, err)
	cal := policyCase()
	cal.Corpus.Roots = []string{"dojo/pilot/fixtures/done"}
	cal.Corpus.Synthetic = true

	require.NoError(t, dojo.ValidateCriteria(criteriaFor("pilot", cal)))
	corpus, err := dojo.BuildCorpus(w.base, cal, handoffconsumer.Reader{})
	require.NoError(t, err)
	require.Len(t, corpus.Manifest.Entries, 4)
	for _, e := range corpus.Manifest.Entries {
		assert.Equal(t, "golden", e.Label)
		assert.False(t, e.WeakLabel)
	}

	for name, mutate := range map[string]func(*domain.DojoCalibration){
		"no roots":         func(c *domain.DojoCalibration) { c.Corpus.Roots = nil },
		"the done root":    func(c *domain.DojoCalibration) { c.Corpus.Roots = []string{"done"} },
		"the refined root": func(c *domain.DojoCalibration) { c.Corpus.Roots = []string{"refined"} },
		"an escape":        func(c *domain.DojoCalibration) { c.Corpus.Roots = []string{"dojo/../done"} },
	} {
		bad := cal
		bad.Corpus.Roots = append([]string(nil), cal.Corpus.Roots...)
		mutate(&bad)
		require.Error(t, dojo.ValidateCriteria(criteriaFor("pilot", bad)), name)
	}

	realCase := policyCase()
	w.packages(t, 2)
	corpus, err = dojo.BuildCorpus(w.base, realCase, handoffconsumer.Reader{})
	require.NoError(t, err)
	for _, e := range corpus.Manifest.Entries {
		assert.Equal(t, "candidate", e.Label, "real packages are never golden by construction")
	}
}

func TestTheLiveLayerIsAValidLayerAndItsFailuresHaveAReason(t *testing.T) {
	cal := policyCase()
	cal.Layer = "live"
	require.NoError(t, dojo.ValidateCriteria(criteriaFor("jev-live", cal)))
	items := []domain.DojoCheckItem{{Label: "live probe authenticates and answers in the documented shape", Passed: false}}
	assert.Equal(t, []dojo.FailureReason{dojo.FailureLiveCheck}, dojo.ClassifyFailures(items))
	assert.Contains(t, dojo.GenerateLesson(domain.DojoCheckResult{Scenario: "jev-live", Items: items}, dojo.ClassifyFailures(items)), "live_check_failed")
}

func TestTheProfileIsPartOfTheTermAndOfTheCaseValidation(t *testing.T) {
	a := baseTerm()
	b := a
	b.Profile = "handoff-v2"
	assert.NotEqual(t, a.ID, dojo.NewTerm(b).ID)
	assert.Contains(t, dojo.NewTerm(b).Text(), "question profile: handoff-v2")
	assert.NotContains(t, a.Text(), "question profile")

	for profile, ok := range map[string]bool{"": true, "handoff-v1": true, "handoff-v2": true, "handoff-atomic-v1": true, "handoff-v9": false} {
		cal := policyCase()
		cal.Profile = profile
		if ok {
			require.NoError(t, dojo.ValidateCriteria(criteriaFor("c", cal)), profile)
		} else {
			require.Error(t, dojo.ValidateCriteria(criteriaFor("c", cal)), profile)
		}
	}
}

func TestTheGateRuleIsValidatedInTheCase(t *testing.T) {
	for rule, ok := range map[string]bool{"": true, "gates-1": true, "gates-2": true, "gates-3": true, "gates-9": false} {
		cal := policyCase()
		cal.GateRule = rule
		if ok {
			require.NoError(t, dojo.ValidateCriteria(criteriaFor("c", cal)), rule)
		} else {
			require.Error(t, dojo.ValidateCriteria(criteriaFor("c", cal)), rule)
		}
	}
}

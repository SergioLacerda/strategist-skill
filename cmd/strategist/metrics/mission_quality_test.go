package metrics

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const goodQualityDoc = `decisions:
  - {id: DEC-001, statement: pin the upstream, status: approved, evidence: [E-1], confidence: medium, gate_answer: "G-1 -> A"}
evidence:
  - {id: E-1, source_ref: docs/a.md, class: explicit, confidence: high}
acceptance_criteria: ["the pin note exists"]
`

func runMissionQuality(t *testing.T, body string, args ...string) (string, error) {
	t.Helper()
	cmd := NewMissionQuality(testDependencies())
	out := &strings.Builder{}
	cmd.SetOut(out)
	cmd.SetIn(strings.NewReader(body))
	cmd.SetArgs(append([]string{"--root", testRoot(t), "--mission", "m-1"}, args...))
	err := cmd.Execute()
	return out.String(), err
}

func TestMissionQualityReportsEveryPredicateForAPassingDocument(t *testing.T) {
	out, err := runMissionQuality(t, goodQualityDoc)
	require.NoError(t, err)
	assert.Contains(t, out, "mission: m-1")
	assert.Contains(t, out, "mission_quality: passed")
	for _, check := range []string{"unsupported_claims", "fact_inference_separation", "traceable_findings", "acceptance_criteria"} {
		assert.Contains(t, out, "check: "+check+" applicable=true passed=true")
	}
	assert.Contains(t, out, "check: unresolved_questions_preserved applicable=false", "not applicable unless the previous open ids are supplied")
	assert.Contains(t, out, "check: source_scope_respected applicable=false", "not applicable unless the scope prefixes are supplied")
}

// The result is advisory: a failed predicate is reported and the command still
// exits 0, so it can never block a mission.
func TestMissionQualityFailedPredicateIsAdvisoryAndListsTheViolation(t *testing.T) {
	doc := strings.Replace(goodQualityDoc, "evidence: [E-1], ", "evidence: [], ", 1)
	out, err := runMissionQuality(t, doc)
	require.NoError(t, err)
	assert.Contains(t, out, "mission_quality: failed")
	assert.Contains(t, out, "check: unsupported_claims applicable=true passed=false")
	assert.Contains(t, out, "violation: decision DEC-001 cites no evidence")
}

func TestMissionQualityChecksPreviouslyOpenDecisionsAndScope(t *testing.T) {
	doc := goodQualityDoc + "previously_open_decision_ids: [DEC-001, DEC-009]\napproved_scope_prefixes: [internal/]\n"
	out, err := runMissionQuality(t, doc)
	require.NoError(t, err)
	assert.Contains(t, out, "violation: previously open decision DEC-009 is missing from the current set")
	assert.Contains(t, out, "check: source_scope_respected applicable=true passed=false")
}

// Without decisions and evidence there is nothing to evaluate; the contract
// says that is not a failure.
func TestMissionQualityWithoutDecisionsOrEvidenceIsNotEvaluated(t *testing.T) {
	out, err := runMissionQuality(t, "acceptance_criteria: [x]\n")
	require.NoError(t, err)
	assert.Contains(t, out, "mission_quality: not_evaluated")
	assert.NotContains(t, out, "check:")
}

func TestMissionQualityRejectsBadInput(t *testing.T) {
	_, err := runMissionQuality(t, "  \n")
	require.ErrorContains(t, err, "standard input is empty")
	_, err = runMissionQuality(t, "decisions: []\nunknown_key: 1\n")
	require.ErrorContains(t, err, `unknown top-level key "unknown_key"`)
	_, err = runMissionQuality(t, "decisions: {id: x}\n")
	require.Error(t, err)
}

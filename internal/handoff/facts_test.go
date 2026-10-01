package handoff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const factsBlockClean = `handoff_policy_facts:
  schema_version: strategist-handoff-policy-facts/v1
  mandatory_constraints: []
  unresolved_questions: []
  forbidden_scope: []
  destructive_operation_possible: false
  security_sensitive_task: false
  informational_only: true
`

const tasksInformational = "## Tasks\n\n- [ ] 1.1 [task_type: documentation_target] write the ADR\n- [ ] 1.2 [analysis_artifact] record evidence\n"
const tasksWithImplementation = tasksInformational + "- [ ] 2.1 [task_type: implementation_handoff] change the code\n"

// writeRefined writes a minimal valid Archivist package; facts is the YAML
// block placed in analysis.md's frontmatter ("" omits it).
func writeRefined(t *testing.T, facts, tasks string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "refined", "m1")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	analysis := "---\nmission_id: m1\nmission_status: gate_analysis_accepted\n" + facts + "---\n\n## mission_objective\nbody\n"
	for name, content := range map[string]string{"analysis.md": analysis, "proposal.md": "proposal\n", "design.md": "design\n", "tasks.md": tasks} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
	}
	return dir
}

func TestExtractRiskSignalsAcceptsAnInformationalPackageAndExplainsWhy(t *testing.T) {
	refined := writeRefined(t, factsBlockClean, tasksInformational)

	got, err := ExtractRiskSignals(refined, "m1", "")

	require.NoError(t, err)
	assert.True(t, got.Signals.InformationalOnly)
	assert.False(t, ResolveArchivistPolicy(got.Signals).Enabled)
	assert.Empty(t, got.Provenance)
}

func TestAnyConcreteRequireFactKeepsALowRiskPackageRequired(t *testing.T) {
	cases := map[string]struct {
		facts, tasks, riskLevel string
		signal                  PolicyPredicate
	}{
		"implementation_handoff task": {factsBlockClean, tasksWithImplementation, "low", PredicateImplementationHandoffPresent},
		"mandatory constraint": {strings.Replace(strings.Replace(factsBlockClean, "mandatory_constraints: []", "mandatory_constraints: [\"no schema change\"]", 1), "informational_only: true", "informational_only: false", 1),
			tasksInformational, "low", PredicateMandatoryConstraintsPresent},
		"unresolved question": {strings.Replace(strings.Replace(factsBlockClean, "unresolved_questions: []", "unresolved_questions: [\"which store?\"]", 1), "informational_only: true", "informational_only: false", 1),
			tasksInformational, "low", PredicateUnresolvedQuestionsPresent},
		"forbidden scope": {strings.Replace(strings.Replace(factsBlockClean, "forbidden_scope: []", "forbidden_scope: [\"internal/**\"]", 1), "informational_only: true", "informational_only: false", 1),
			tasksInformational, "low", PredicateForbiddenScopePresent},
		"destructive operation": {strings.Replace(strings.Replace(factsBlockClean, "destructive_operation_possible: false", "destructive_operation_possible: true", 1), "informational_only: true", "informational_only: false", 1),
			tasksInformational, "low", PredicateDestructiveOperationPossible},
		"security sensitive": {strings.Replace(strings.Replace(factsBlockClean, "security_sensitive_task: false", "security_sensitive_task: true", 1), "informational_only: true", "informational_only: false", 1),
			tasksInformational, "low", PredicateSecuritySensitiveTask},
	}
	for name, tc := range cases {
		refined := writeRefined(t, tc.facts, tc.tasks)
		if name == "implementation_handoff task" {
			_, err := ExtractRiskSignals(refined, "m1", tc.riskLevel)
			require.ErrorContains(t, err, "handoff_policy_facts_contradictory", "an informational declaration cannot coexist with an implementation task")
			refined = writeRefined(t, strings.Replace(factsBlockClean, "informational_only: true", "informational_only: false", 1), tc.tasks)
		}

		got, err := ExtractRiskSignals(refined, "m1", tc.riskLevel)

		require.NoError(t, err, name)
		assert.True(t, ResolveArchivistPolicy(got.Signals).Enabled, name)
		require.NotEmpty(t, got.Provenance, name)
		assert.Equal(t, string(tc.signal), got.Provenance[0].Signal, name)
	}
}

func TestCoarseRiskCanOnlyMakeTheResultStricter(t *testing.T) {
	refined := writeRefined(t, factsBlockClean, tasksInformational)

	for _, level := range []string{"medium", "high"} {
		got, err := ExtractRiskSignals(refined, "m1", level)
		require.NoError(t, err)
		assert.False(t, got.Signals.InformationalOnly, level)
		assert.True(t, ResolveArchivistPolicy(got.Signals).Enabled, level)
	}
	low, err := ExtractRiskSignals(refined, "m1", "low")
	require.NoError(t, err)
	assert.False(t, ResolveArchivistPolicy(low.Signals).Enabled, "low adds nothing")

	_, err = ExtractRiskSignals(refined, "m1", "catastrophic")
	require.ErrorContains(t, err, "handoff_risk_level_unknown")
}

func TestExtractRiskSignalsRejectsMissingUnknownAndContradictoryFacts(t *testing.T) {
	missing := strings.Replace(factsBlockClean, "  security_sensitive_task: false\n", "", 1)
	nullList := strings.Replace(factsBlockClean, "unresolved_questions: []", "unresolved_questions:", 1)
	unknownKey := factsBlockClean + "  risk_is_low: true\n"
	badVersion := strings.Replace(factsBlockClean, "facts/v1", "facts/v9", 1)
	notABool := strings.Replace(factsBlockClean, "security_sensitive_task: false", "security_sensitive_task: maybe", 1)
	contradictory := strings.Replace(factsBlockClean, "unresolved_questions: []", "unresolved_questions: [\"open?\"]", 1)
	for name, tc := range map[string]struct{ facts, want string }{
		"no block":               {"", "handoff_policy_facts_missing"},
		"missing field":          {missing, "handoff_policy_facts_missing"},
		"null list":              {nullList, "handoff_policy_facts_missing"},
		"unknown key":            {unknownKey, "handoff_policy_facts_invalid"},
		"wrong schema version":   {badVersion, "handoff_policy_facts_invalid"},
		"string instead of bool": {notABool, "handoff_policy_facts_invalid"},
		"informational + open":   {contradictory, "handoff_policy_facts_contradictory"},
	} {
		_, err := ExtractRiskSignals(writeRefined(t, tc.facts, tasksInformational), "m1", "")
		require.ErrorContains(t, err, tc.want, name)
	}
}

func TestPackageDigestIgnoresStatusButNotContent(t *testing.T) {
	refined := writeRefined(t, factsBlockClean, tasksInformational)
	before, err := PackageDigest(refined)
	require.NoError(t, err)

	analysis := filepath.Join(refined, "analysis.md")
	raw, err := os.ReadFile(analysis)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(analysis, []byte(strings.Replace(string(raw), "gate_analysis_accepted", "sniper_running", 1)), 0o644))
	sameStatus, err := PackageDigest(refined)
	require.NoError(t, err)
	assert.Equal(t, before, sameStatus, "advancing mission_status is not a new package revision")

	for _, change := range []struct{ file, from, to string }{
		{"tasks.md", "ADR", "ADR (amended)"}, {"design.md", "design", "design v2"}, {"proposal.md", "proposal", "proposal v2"},
		{"analysis.md", "body", "different body"}, {"analysis.md", "informational_only: true", "informational_only: false"},
	} {
		path := filepath.Join(refined, change.file)
		original, err := os.ReadFile(path)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, []byte(strings.Replace(string(original), change.from, change.to, 1)), 0o644))
		changed, err := PackageDigest(refined)
		require.NoError(t, err)
		assert.NotEqual(t, before, changed, "%s: %q", change.file, change.from)
		require.NoError(t, os.WriteFile(path, original, 0o644))
	}
}

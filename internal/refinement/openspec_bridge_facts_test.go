package refinement

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func validFacts() map[string]any {
	return map[string]any{
		"schema_version":                 handoff.PolicyFactsSchemaVersion,
		"mandatory_constraints":          []any{"no source edits"},
		"unresolved_questions":           []any{},
		"forbidden_scope":                []any{"internal/**"},
		"destructive_operation_possible": false,
		"security_sensitive_task":        false,
		"informational_only":             false,
	}
}

// factsFixture lays out a pending analysis and a completed change ready to publish.
func factsFixture(t *testing.T, mission string) (OpenSpecInput, string) {
	t.Helper()
	project := t.TempDir()
	base := filepath.Join(project, ".analysis")
	runtime := filepath.Join(project, ".strategist", "openspec")
	pending := filepath.Join(base, "pending", mission+"-analysis.md")
	changeDir := filepath.Join(runtime, "changes", "change-1")
	require.NoError(t, os.MkdirAll(filepath.Dir(pending), 0o755))
	require.NoError(t, os.MkdirAll(changeDir, 0o755))
	require.NoError(t, os.WriteFile(pending, []byte("---\nschema_version: strategist-ranger-discovery/v1\nmission_id: "+mission+"\nmission_status: archivist_pending\nsources_consulted: []\n---\n\n## mission_objective\nobjective\n## known_facts\nfacts\n## confidence_summary\nsummary\n## handoff\nhandoff\n"), 0o644))
	for name, content := range map[string]string{"proposal.md": "# proposal\n", "design.md": "# design\n", "tasks.md": "- [ ] 1.1 [task_type: analysis_artifact] record the evidence\n"} {
		require.NoError(t, os.WriteFile(filepath.Join(changeDir, name), []byte(content), 0o644))
	}
	return OpenSpecInput{
		MissionID: mission, BasePath: base, RuntimeRoot: runtime, ChangeID: "change-1", PendingAnalysisPath: pending,
		RecordConfidence: noopConfidenceRecorder,
	}, base
}

func TestNormalizeOpenSpecPublishesTheDeclaredHandoffFactsAsAnEvaluablePackage(t *testing.T) {
	input, base := factsFixture(t, "m-facts")
	input.HandoffFacts = validFacts()

	result, err := NormalizeOpenSpec(input)

	require.NoError(t, err)
	published, err := os.ReadFile(filepath.Join(result.RefinedPath, "analysis.md"))
	require.NoError(t, err)
	assert.Contains(t, string(published), "handoff_policy_facts:")
	signals, err := handoff.ExtractRiskSignals(filepath.Join(base, "refined", "m-facts"), "m-facts", "")
	require.NoError(t, err, "a published package passes the production extractor without hand-editing")
	assert.True(t, signals.Signals.MandatoryConstraintsPresent)
	assert.True(t, signals.Signals.ForbiddenScopePresent)
	assert.True(t, handoff.ResolveArchivistPolicy(signals.Signals).Enabled)
}

func TestNormalizeOpenSpecDoesNotConfuseRangerFactsWithArchivistFacts(t *testing.T) {
	input, _ := factsFixture(t, "m-ranger-facts")
	raw, err := os.ReadFile(input.PendingAnalysisPath)
	require.NoError(t, err)
	rangerFacts := "ranger_handoff_policy_facts:\n  schema_version: strategist-ranger-handoff-policy-facts/v1\n  require_recall: false\n  require_boundary: false\n  require_classification: false\n  require_verdict: false\n  informational_only: true\n"
	updated := strings.Replace(string(raw), "\n---\n", "\n"+rangerFacts+"---\n", 1)
	require.NoError(t, os.WriteFile(input.PendingAnalysisPath, []byte(updated), 0o644))
	input.HandoffFacts = validFacts()

	result, err := NormalizeOpenSpec(input)

	require.NoError(t, err)
	published, err := os.ReadFile(filepath.Join(result.RefinedPath, "analysis.md"))
	require.NoError(t, err)
	assert.Contains(t, string(published), "ranger_handoff_policy_facts:")
	assert.Contains(t, string(published), "handoff_policy_facts:")
}

func TestNormalizeOpenSpecWithoutFactsPublishesNoBlockAndTheExtractorReportsIt(t *testing.T) {
	input, base := factsFixture(t, "m-nofacts")

	result, err := NormalizeOpenSpec(input)

	require.NoError(t, err)
	published, err := os.ReadFile(filepath.Join(result.RefinedPath, "analysis.md"))
	require.NoError(t, err)
	assert.NotContains(t, string(published), handoff.PolicyFactsKey)
	_, err = handoff.ExtractRiskSignals(filepath.Join(base, "refined", "m-nofacts"), "m-nofacts", "")
	require.ErrorContains(t, err, "handoff_policy_facts_missing")
}

func TestNormalizeOpenSpecRejectsInvalidFactsBeforePublishing(t *testing.T) {
	cases := map[string]func(map[string]any){
		"missing field":    func(f map[string]any) { delete(f, "security_sensitive_task") },
		"unknown key":      func(f map[string]any) { f["risk_is_low"] = true },
		"wrong version":    func(f map[string]any) { f["schema_version"] = "v9" },
		"null list":        func(f map[string]any) { f["forbidden_scope"] = nil },
		"non-boolean flag": func(f map[string]any) { f["informational_only"] = "maybe" },
	}
	for name, mutate := range cases {
		input, base := factsFixture(t, "m-bad")
		facts := validFacts()
		mutate(facts)
		input.HandoffFacts = facts

		_, err := NormalizeOpenSpec(input)

		require.ErrorContains(t, err, "handoff_policy_facts", name)
		assert.NoDirExists(t, filepath.Join(base, "refined", "m-bad"), "%s: nothing is published", name)
		assert.FileExists(t, input.PendingAnalysisPath, "%s: the pending analysis is untouched", name)
	}
}

func TestNormalizeOpenSpecRefusesAPendingAnalysisThatAlreadyDeclaresFacts(t *testing.T) {
	input, _ := factsFixture(t, "m-dup")
	raw, err := os.ReadFile(input.PendingAnalysisPath)
	require.NoError(t, err)
	encoded, err := yaml.Marshal(map[string]any{handoff.PolicyFactsKey: validFacts()})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(input.PendingAnalysisPath, []byte(string(raw)[:len("---\n")]+string(encoded)+string(raw)[len("---\n"):]), 0o644))
	input.HandoffFacts = validFacts()

	_, err = NormalizeOpenSpec(input)

	require.ErrorContains(t, err, "already carries handoff_policy_facts")
}

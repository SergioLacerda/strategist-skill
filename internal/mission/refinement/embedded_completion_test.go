package refinement

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const embeddedFacts = `schema_version: strategist-handoff-policy-facts/v1
mandatory_constraints: [keep the existing paths]
unresolved_questions: []
forbidden_scope: [source changes]
destructive_operation_possible: false
security_sensitive_task: false
informational_only: false`

func embeddedCompletion(facts, tasks string) string {
	return "---\nhandoff_policy_facts:\n" + indentFacts(facts) + "---\n\n" +
		"## proposal.md\n\n### Why\n\nreason\n\n" +
		"## design.md\n\n### Context\n\ncontext\n\n" +
		"## tasks.md\n\n" + tasks + "\n"
}

func indentFacts(facts string) string {
	lines := strings.Split(facts, "\n")
	for i, line := range lines {
		lines[i] = "  " + line
	}
	return strings.Join(lines, "\n") + "\n"
}

const embeddedTask = "- [ ] 1.1 [task_type: implementation_handoff] change the thing and verify the test passes"

func TestParseEmbeddedCompletionSplitsThreeDocumentsAndTheTypedFacts(t *testing.T) {
	parsed, err := ParseEmbeddedCompletion(embeddedCompletion(embeddedFacts, embeddedTask))

	require.NoError(t, err)
	assert.Equal(t, "### Why\n\nreason\n", string(parsed.Documents["proposal.md"]))
	assert.Equal(t, "### Context\n\ncontext\n", string(parsed.Documents["design.md"]))
	assert.Equal(t, embeddedTask+"\n", string(parsed.Documents["tasks.md"]))
	assert.Equal(t, false, parsed.HandoffFacts["informational_only"])
	assert.Equal(t, "strategist-handoff-policy-facts/v1", parsed.HandoffFacts["schema_version"])
}

func TestParseEmbeddedCompletionFailsClosed(t *testing.T) {
	valid := embeddedCompletion(embeddedFacts, embeddedTask)
	cases := map[string]string{
		"no frontmatter":            strings.TrimPrefix(valid, "---\nhandoff_policy_facts:\n"+indentFacts(embeddedFacts)+"---\n\n"),
		"missing document":          strings.Replace(valid, "## design.md\n\n### Context\n\ncontext\n\n", "", 1),
		"duplicated document":       valid + "## proposal.md\n\nagain\n",
		"unexpected level-2":        strings.Replace(valid, "### Why", "## Why", 1),
		"text before first section": strings.Replace(valid, "---\n\n## proposal.md", "---\n\nintro\n\n## proposal.md", 1),
		"empty document":            strings.Replace(valid, "### Context\n\ncontext\n", "", 1),
		"unknown frontmatter key":   strings.Replace(valid, "---\n\n## proposal.md", "extra: 1\n---\n\n## proposal.md", 1),
		"incomplete facts":          embeddedCompletion("schema_version: strategist-handoff-policy-facts/v1\ninformational_only: false", embeddedTask),
		"wrong facts schema":        embeddedCompletion(strings.Replace(embeddedFacts, "policy-facts/v1", "policy-facts/v0", 1), embeddedTask),
	}
	for name, result := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := ParseEmbeddedCompletion(result)
			require.Error(t, err)
		})
	}
}

func TestParseEmbeddedCompletionRejectsAContainmentViolation(t *testing.T) {
	cases := map[string]string{
		"plan saved under docs/plans": "- [ ] 1.1 [task_type: documentation_target] Save the plan to `docs/plans/2026-10-10-x.md` and verify the file exists",
		"commit step":                 "- [ ] 1.1 [task_type: implementation_handoff] Run `git commit -m x` and verify the log",
	}
	for name, tasks := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := ParseEmbeddedCompletion(embeddedCompletion(embeddedFacts, tasks))
			require.ErrorContains(t, err, "embedded_containment_violation")
		})
	}
}

func TestParseEmbeddedCompletionAllowsAMentionOfDocsPlans(t *testing.T) {
	tasks := "- [ ] 1.1 [task_type: implementation_handoff] Confirm nothing is read from docs/plans/ any more and verify the grep is empty"

	_, err := ParseEmbeddedCompletion(embeddedCompletion(embeddedFacts, tasks))

	require.NoError(t, err)
}

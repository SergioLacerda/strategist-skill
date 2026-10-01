package handoff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const rangerFactsArtifact = `---
schema_version: strategist-ranger-discovery/v1
mission_id: m-ranger
mission_status: ranger_pending
sources_consulted: []
ranger_handoff_policy_facts:
  schema_version: strategist-ranger-handoff-policy-facts/v1
  require_recall: true
  require_boundary: true
  require_classification: true
  require_verdict: true
  informational_only: false
discovery_subtype: evaluation
evaluation_verdict: partially_implemented
---

## mission_objective
objective
## known_facts
- id: F-1
  statement: fact
## confidence_summary
summary
## handoff
handoff
`

func rangerArtifactFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "m-ranger-analysis.md")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestReadRangerPolicyFactsRejectsMissingUnknownAndContradictoryFacts(t *testing.T) {
	for name, replacement := range map[string]string{
		"missing block": "ranger_handoff_policy_facts: null\n",
		"unknown key":   "ranger_handoff_policy_facts:\n  schema_version: strategist-ranger-handoff-policy-facts/v1\n  unknown: true\n",
		"contradictory": "ranger_handoff_policy_facts:\n  schema_version: strategist-ranger-handoff-policy-facts/v1\n  require_recall: true\n  require_boundary: false\n  require_classification: false\n  require_verdict: false\n  informational_only: true\n",
		"ambiguous":     "ranger_handoff_policy_facts:\n  schema_version: strategist-ranger-handoff-policy-facts/v1\n  require_recall: false\n  require_boundary: false\n  require_classification: false\n  require_verdict: false\n  informational_only: false\n",
	} {
		t.Run(name, func(t *testing.T) {
			content := replaceRangerFacts(rangerFactsArtifact, replacement)
			_, err := ReadRangerPolicyFacts(rangerArtifactFixture(t, content), "m-ranger")
			require.Error(t, err)
		})
	}
}

func replaceRangerFacts(content, replacement string) string {
	start := strings.Index(content, "ranger_handoff_policy_facts:")
	end := strings.Index(content[start:], "discovery_subtype:")
	if start < 0 || end < 0 {
		return content
	}
	end += start
	return content[:start] + replacement + content[end:]
}

func TestRangerPolicyFactsDeriveOnlyDeclaredChallengeTypes(t *testing.T) {
	facts, err := ReadRangerPolicyFacts(rangerArtifactFixture(t, rangerFactsArtifact), "m-ranger")
	require.NoError(t, err)
	policy, err := RangerToArchivistPolicyForFacts(facts)
	require.NoError(t, err)
	require.Equal(t, []string{ChallengeRecall, ChallengeBoundary, ChallengeClassification, ChallengeVerdict}, policy.RequiredTypes)
	require.True(t, policy.Enabled)

	facts.RequireVerdict = boolPointer(false)
	withoutVerdict, err := RangerToArchivistPolicyForFacts(facts)
	require.NoError(t, err)
	require.NotEqual(t, PolicyIdentity(policy), PolicyIdentity(withoutVerdict))
}

func boolPointer(value bool) *bool { return &value }

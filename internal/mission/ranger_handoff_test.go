package mission

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/require"
)

const rangerHandoffArtifact = `---
schema_version: strategist-ranger-discovery/v1
mission_id: m-ranger
mission_status: ranger_done
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

func rangerHandoffFixture(t *testing.T, content string) (string, string) {
	t.Helper()
	root := t.TempDir()
	basePath := filepath.Join(root, "analysis")
	artifactPath := filepath.Join(basePath, "pending", "m-ranger-analysis.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(artifactPath), 0o755))
	require.NoError(t, os.WriteFile(artifactPath, []byte(content), 0o644))
	return root, basePath
}

func rangerChallenges() []handoff.Challenge {
	return []handoff.Challenge{
		{ID: "HC-RECALL", Type: handoff.ChallengeRecall},
		{ID: "HC-BOUNDARY", Type: handoff.ChallengeBoundary},
		{ID: "HC-CLASSIFICATION", Type: handoff.ChallengeClassification},
		{ID: "HC-VERDICT", Type: handoff.ChallengeVerdict},
	}
}

func TestRangerHandoffFailsThenPassesAndConsumesBeforeArchivist(t *testing.T) {
	root, basePath := rangerHandoffFixture(t, rangerHandoffArtifact)

	failed, err := EvaluateRangerToArchivist(root, filepath.Join(basePath, "pending", "m-ranger-analysis.md"), "m-ranger", RangerHandoffInput{})
	require.NoError(t, err)
	require.Equal(t, handoff.OutcomeFailed, failed.Outcome.Result)
	_, err = AuthorizeRangerToArchivist(root, basePath, "m-ranger")
	require.ErrorContains(t, err, "handoff_outcome_failed")

	passed, err := EvaluateRangerToArchivist(root, filepath.Join(basePath, "pending", "m-ranger-analysis.md"), "m-ranger", RangerHandoffInput{Challenges: rangerChallenges()})
	require.NoError(t, err)
	require.Equal(t, handoff.OutcomePassed, passed.Outcome.Result)
	consumed, err := AuthorizeRangerToArchivist(root, basePath, "m-ranger")
	require.NoError(t, err)
	require.Equal(t, passed.Outcome.Integrity, consumed.Integrity)
	_, err = AuthorizeRangerToArchivist(root, basePath, "m-ranger")
	require.ErrorContains(t, err, "handoff_outcome_replayed")
}

func TestRangerHandoffSkipsOnlyWhenTypedFactsAuthorizeIt(t *testing.T) {
	content := replaceRangerHandoffFacts(rangerHandoffArtifact, `ranger_handoff_policy_facts:
  schema_version: strategist-ranger-handoff-policy-facts/v1
  require_recall: false
  require_boundary: false
  require_classification: false
  require_verdict: false
  informational_only: true
`)
	root, basePath := rangerHandoffFixture(t, content)

	require.NoError(t, EnsureRangerToArchivistOutcome(root, basePath, "m-ranger"))
	outcome, err := AuthorizeRangerToArchivist(root, basePath, "m-ranger")
	require.NoError(t, err)
	require.Equal(t, handoff.OutcomeSkipped, outcome.Result)
}

func TestRangerHandoffTelemetryDoesNotIncludeArtifactOrAnswers(t *testing.T) {
	root, basePath := rangerHandoffFixture(t, rangerHandoffArtifact)
	sink := &rangerHandoffCapture{}

	_, err := EvaluateRangerToArchivistWithTelemetry(context.Background(), root, filepath.Join(basePath, "pending", "m-ranger-analysis.md"), "m-ranger", RangerHandoffInput{Challenges: rangerChallenges()}, sink, "run-1")
	require.NoError(t, err)
	events := sink.eventsSnapshot()
	require.Len(t, events, 1)
	require.Equal(t, telemetry.RangerToArchivistEventName, events[0].Name)
	require.Equal(t, handoff.OutcomePassed, events[0].Attributes[telemetry.AttrHandoffOutcome])
	require.NotContains(t, events[0].Attributes, telemetry.AttrArtifact)
	require.NotContains(t, events[0].Attributes, telemetry.AttrArtifactPath)
	require.NotContains(t, events[0].Attributes, telemetry.AttrInvocationEvidence)
}

func TestRangerHandoffCompletionConsumesNoAttemptWhenChallengeRequired(t *testing.T) {
	root, basePath := rangerHandoffFixture(t, rangerHandoffArtifact)
	artifact := filepath.Join(basePath, "pending", "m-ranger-analysis.md")

	require.NoError(t, EnsureRangerToArchivistOutcome(root, basePath, "m-ranger"))

	next, err := handoff.NewOutcomeStore(root).NextAttemptFor("m-ranger", handoff.TransitionRangerToArchivist)
	require.NoError(t, err)
	require.Equal(t, 1, next, "normalization must not consume a bounded attempt: no answers can exist yet")
	_, err = AuthorizeRangerToArchivist(root, basePath, "m-ranger")
	require.ErrorContains(t, err, "handoff_outcome_missing", "the Archivist boundary stays closed until an explicit evaluation")

	failed, err := EvaluateRangerToArchivist(root, artifact, "m-ranger", RangerHandoffInput{})
	require.NoError(t, err)
	require.Equal(t, 1, failed.Outcome.Attempt)
	require.Equal(t, handoff.OutcomeFailed, failed.Outcome.Result)
	passed, err := EvaluateRangerToArchivist(root, artifact, "m-ranger", RangerHandoffInput{Challenges: rangerChallenges()})
	require.NoError(t, err)
	require.Equal(t, 2, passed.Outcome.Attempt, "both bounded attempts remain available for real answers")
	require.Equal(t, handoff.OutcomePassed, passed.Outcome.Result)
}

func TestRangerHandoffCompletionEmitsAwaitingEventWhenChallengeRequired(t *testing.T) {
	root, basePath := rangerHandoffFixture(t, rangerHandoffArtifact)
	sink := &rangerHandoffCapture{}

	require.NoError(t, EnsureRangerToArchivistOutcomeWithTelemetry(context.Background(), root, basePath, "m-ranger", sink, "m-ranger"))

	events := sink.eventsSnapshot()
	require.Len(t, events, 1)
	require.Equal(t, telemetry.RangerToArchivistEventName, events[0].Name)
	require.Equal(t, RangerOutcomeAwaitingChallenge, events[0].Attributes[telemetry.AttrHandoffOutcome])
	require.Equal(t, true, events[0].Attributes[telemetry.AttrHandoffRequired])
	require.Equal(t, 0, events[0].Attributes[telemetry.AttrHandoffAttempt])
	require.Equal(t, telemetry.SeverityInfo, events[0].SeverityNumber, "waiting for answers is not a failure")
	require.NotContains(t, events[0].Attributes, telemetry.AttrReason)
}

func replaceRangerHandoffFacts(content, replacement string) string {
	start := 0
	for i, line := range splitRangerHandoffLines(content) {
		if line == "ranger_handoff_policy_facts:" {
			start = i
			break
		}
	}
	lines := splitRangerHandoffLines(content)
	end := start + 1
	for end < len(lines) && (lines[end] == "" || lines[end][0] == ' ' || lines[end][0] == '\t') {
		end++
	}
	return joinRangerHandoffLines(append(append(append([]string{}, lines[:start]...), splitRangerHandoffLines(replacement)...), lines[end:]...))
}

func splitRangerHandoffLines(value string) []string {
	return strings.Split(strings.TrimSuffix(value, "\n"), "\n")
}

func joinRangerHandoffLines(lines []string) string {
	result := ""
	for _, line := range lines {
		result += line + "\n"
	}
	return result
}

type rangerHandoffCapture struct {
	mu     sync.Mutex
	events []telemetry.Event
}

func (c *rangerHandoffCapture) Emit(_ context.Context, event telemetry.Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, event)
	return nil
}

func (c *rangerHandoffCapture) eventsSnapshot() []telemetry.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]telemetry.Event(nil), c.events...)
}

package mission

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/stretchr/testify/require"
)

func approvedDelegation(subject string) *handoff.Delegation {
	return &handoff.Delegation{
		Provider: "jev", Model: "jev-1.13.0", BindingDigest: "sha256:binding", Capability: "handoff.validate",
		Criterion: "handoff_acceptable", Subject: subject, Checks: handoff.DelegableChecks(),
		Confidence: 0.95, Threshold: 0.90, InputTokens: 100, OutputTokens: 10,
	}
}

func TestRangerEvaluationRecordsAValidDelegationAndStillAuthorizes(t *testing.T) {
	root, basePath := rangerHandoffFixture(t, rangerHandoffArtifact)
	artifact := filepath.Join(basePath, "pending", "m-ranger-analysis.md")
	digest, err := handoff.RangerArtifactDigest(artifact)
	require.NoError(t, err)

	result, err := EvaluateRangerToArchivist(root, artifact, "m-ranger", RangerHandoffInput{Challenges: rangerChallenges(), Delegation: approvedDelegation(digest)})
	require.NoError(t, err)
	require.Equal(t, handoff.OutcomePassed, result.Outcome.Result, "the deterministic challenge still decides the result")
	require.NotNil(t, result.Outcome.Delegation)
	require.NoError(t, result.Outcome.VerifyIntegrity())

	consumed, err := AuthorizeRangerToArchivist(root, basePath, "m-ranger")
	require.NoError(t, err)
	require.Equal(t, result.Outcome.Integrity, consumed.Integrity)
}

func TestAStaleDelegationIsDroppedAndTheHandoffProceedsOnMain(t *testing.T) {
	root, basePath := rangerHandoffFixture(t, rangerHandoffArtifact)
	artifact := filepath.Join(basePath, "pending", "m-ranger-analysis.md")

	result, err := EvaluateRangerToArchivist(root, artifact, "m-ranger", RangerHandoffInput{Challenges: rangerChallenges(), Delegation: approvedDelegation("sha256:an-older-revision")})
	require.NoError(t, err, "a provider result can never block a handoff")
	require.Equal(t, handoff.OutcomePassed, result.Outcome.Result)
	require.Nil(t, result.Outcome.Delegation)
}

func TestAutomaticRangerSkipRecordsTheDelegationFromTheCallback(t *testing.T) {
	root, basePath := rangerHandoffFixture(t, replaceRangerHandoffFacts(rangerHandoffArtifact, `ranger_handoff_policy_facts:
  schema_version: strategist-ranger-handoff-policy-facts/v1
  require_recall: false
  require_boundary: false
  require_classification: false
  require_verdict: false
  informational_only: true
`))
	artifact := filepath.Join(basePath, "pending", "m-ranger-analysis.md")
	want, err := handoff.RangerArtifactDigest(artifact)
	require.NoError(t, err)
	var subject string

	err = EnsureRangerToArchivistOutcomeWithDelegation(context.Background(), root, basePath, "m-ranger", nil, "m-ranger", func(s string) *handoff.Delegation {
		subject = s
		return approvedDelegation(s)
	})
	require.NoError(t, err)
	require.Equal(t, want, subject, "the callback receives the digest of the revision being skipped")

	outcome, err := AuthorizeRangerToArchivist(root, basePath, "m-ranger")
	require.NoError(t, err)
	require.Equal(t, handoff.OutcomeSkipped, outcome.Result)
	require.NotNil(t, outcome.Delegation)
}

func TestTheCallbackIsNotCalledWhenAChallengeIsRequired(t *testing.T) {
	root, basePath := rangerHandoffFixture(t, rangerHandoffArtifact)
	called := false

	err := EnsureRangerToArchivistOutcomeWithDelegation(context.Background(), root, basePath, "m-ranger", nil, "m-ranger", func(string) *handoff.Delegation {
		called = true
		return nil
	})
	require.NoError(t, err)
	require.False(t, called, "no outcome is recorded yet, so there is nothing to attach a delegation to")
}

func TestArchivistEvaluationRecordsADelegationBoundToThePackageDigest(t *testing.T) {
	f := newRecordFixture(t, "rec-delegated", recordFacts)
	digest, err := handoff.PackageDigest(filepath.Join(f.basePath, "refined", f.id))
	require.NoError(t, err)
	input := recordChallenges()
	input.Delegation = approvedDelegation(digest)

	evaluation, _, _, err := RecordArchivistHandoff(f.root, f.basePath, f.engine, input)
	require.NoError(t, err)
	require.Equal(t, handoff.OutcomePassed, evaluation.Outcome.Result)
	require.NotNil(t, evaluation.Outcome.Delegation)
	require.Equal(t, digest, evaluation.Outcome.Delegation.Subject)
}

func TestArchivistEvaluationDropsAStaleDelegation(t *testing.T) {
	f := newRecordFixture(t, "rec-stale", recordFacts)
	input := recordChallenges()
	input.Delegation = approvedDelegation("sha256:stale")

	evaluation, _, _, err := RecordArchivistHandoff(f.root, f.basePath, f.engine, input)
	require.NoError(t, err)
	require.Equal(t, handoff.OutcomePassed, evaluation.Outcome.Result)
	require.Nil(t, evaluation.Outcome.Delegation)
}

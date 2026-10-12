package refinement

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const embeddedPending = `---
schema_version: strategist-ranger-discovery/v1
mission_id: m-emb
mission_status: ranger_done
sources_consulted: []
---

## mission_objective
objective
## known_facts
facts
## confidence_summary
summary
## handoff
handoff
`

type embeddedFixture struct {
	base, pending string
	claims        []domain.ConfidenceClaim
	publications  []PackagePublication
}

func newEmbeddedFixture(t *testing.T) *embeddedFixture {
	t.Helper()
	base := filepath.Join(t.TempDir(), ".analysis")
	pending := filepath.Join(base, "pending", "m-emb-analysis.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(pending), 0o755))
	require.NoError(t, os.WriteFile(pending, []byte(embeddedPending), 0o644))
	return &embeddedFixture{base: base, pending: pending}
}

func (f *embeddedFixture) input(t *testing.T, tasks string) EmbeddedInput {
	t.Helper()
	parsed, err := ParseEmbeddedCompletion(embeddedCompletion(embeddedFacts, tasks))
	require.NoError(t, err)
	return EmbeddedInput{
		MissionID: "m-emb", BasePath: f.base, PendingAnalysisPath: f.pending,
		ProviderID: "writing-plans", RequestID: "inv_0123456789abcdef", Completion: parsed,
		RecordConfidence: func(claim domain.ConfidenceClaim, _ []domain.Evidence) error {
			f.claims = append(f.claims, claim)
			return nil
		},
		RecordPublication: func(p PackagePublication) error {
			f.publications = append(f.publications, p)
			return nil
		},
	}
}

func TestPublishEmbeddedWritesTheCanonicalPackageAndPromotesPending(t *testing.T) {
	f := newEmbeddedFixture(t)

	result, err := PublishEmbedded(f.input(t, embeddedTask))

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(f.base, "refined", "m-emb"), result.RefinedPath)
	assertFile(t, filepath.Join(result.RefinedPath, "analysis.md"),
		"mission_status: archivist_done", "provider: writing-plans", "provider_change_id: inv_0123456789abcdef",
		"provider_runtime: embedded", "handoff_policy_facts:")
	assertFile(t, filepath.Join(result.RefinedPath, "proposal.md"), "### Why")
	assertFile(t, filepath.Join(result.RefinedPath, "tasks.md"), "[task_type: implementation_handoff]")
	_, err = os.Stat(f.pending)
	require.ErrorIs(t, err, os.ErrNotExist, "the pending analysis is promoted")
	require.Len(t, f.claims, 1)
	assert.Equal(t, "archivist", f.claims[0].Agent)
	assert.Equal(t, "archivist-embedded-normalization", f.claims[0].CorrelationKey)
	require.Len(t, f.publications, 1)
	assert.Equal(t, "inv_0123456789abcdef", f.publications[0].ProviderChangeID)
}

func TestPublishEmbeddedRejectsAnUnclassifiedTaskAndLeavesPendingInPlace(t *testing.T) {
	f := newEmbeddedFixture(t)

	_, err := PublishEmbedded(f.input(t, "- [ ] 1.1 change the thing and verify it"))

	require.ErrorContains(t, err, "classification")
	_, statErr := os.Stat(filepath.Join(f.base, "refined", "m-emb"))
	require.ErrorIs(t, statErr, os.ErrNotExist, "nothing is published")
	_, statErr = os.Stat(f.pending)
	require.NoError(t, statErr, "the pending analysis stays in place")
	assert.Empty(t, f.claims, "no claim is recorded for a package that failed validation")
}

func TestPublishEmbeddedRequiresTheFactsWhenThereIsADocumentationTarget(t *testing.T) {
	f := newEmbeddedFixture(t)
	input := f.input(t, "- [ ] 1.1 [task_type: documentation_target] Write `docs/adr/0100-x.md` and verify it exists")
	input.Completion.HandoffFacts = nil

	_, err := PublishEmbedded(input)

	require.ErrorContains(t, err, "handoff_policy_facts")
	_, statErr := os.Stat(filepath.Join(f.base, "refined", "m-emb"))
	require.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestPublishEmbeddedIsIdempotentForAnIdenticalReplayAndFailsClosedOnConflict(t *testing.T) {
	f := newEmbeddedFixture(t)
	_, err := PublishEmbedded(f.input(t, embeddedTask))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(f.pending, []byte(embeddedPending), 0o644))

	_, err = PublishEmbedded(f.input(t, embeddedTask))
	require.NoError(t, err, "an identical package is kept")
	assert.Len(t, f.publications, 1, "a replay records no second publication")

	require.NoError(t, os.WriteFile(f.pending, []byte(embeddedPending), 0o644))
	_, err = PublishEmbedded(f.input(t, "- [ ] 1.1 [task_type: implementation_handoff] a different plan and verify it"))
	require.ErrorContains(t, err, "conflicts")
}

func TestPublishEmbeddedRejectsAMalformedMissionID(t *testing.T) {
	f := newEmbeddedFixture(t)
	input := f.input(t, embeddedTask)
	input.MissionID = "../escape"

	_, err := PublishEmbedded(input)

	require.Error(t, err)
}

func TestPublishEmbeddedRecoversAfterACrashBetweenPublicationAndFinalization(t *testing.T) {
	f := newEmbeddedFixture(t)
	input := f.input(t, embeddedTask)
	_, err := PublishEmbedded(input)
	require.NoError(t, err)
	_, statErr := os.Stat(f.pending)
	require.ErrorIs(t, statErr, os.ErrNotExist, "the first attempt promoted the pending analysis")

	result, err := PublishEmbedded(input)

	require.NoError(t, err, "the same completion is recognized as already published")
	assert.Equal(t, filepath.Join(f.base, "refined", "m-emb"), result.RefinedPath)
	assert.Len(t, f.publications, 1, "no second publication is recorded")
	assert.Len(t, f.claims, 1, "no second confidence claim is recorded")

	other := f.input(t, "- [ ] 1.1 [task_type: implementation_handoff] a different plan and verify it")
	other.RequestID = "inv_fedcba9876543210"
	_, err = PublishEmbedded(other)
	require.ErrorContains(t, err, "conflicts", "a different completion never overwrites the published package")
}

package telemetry

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func declaredClaim(id, agent, kind string, percent int) domain.ConfidenceClaim {
	return domain.ConfidenceClaim{ID: id, Statement: "s " + id, Agent: agent, CorrelationKey: "k-" + id, ClaimKind: kind, ConfidencePercent: percent}
}

func persistedRecord(id, agent, kind string, percent int, level, status string) ConfidenceRecord {
	return ConfidenceRecord{ClaimID: id, Agent: agent, CorrelationKey: "k-" + id, ClaimKind: kind, ConfidencePercent: percent, ConfidenceLevel: level, CoverageStatus: status}
}

func summaryOf(claims ...domain.ConfidenceClaim) domain.ConfidenceSummary {
	return domain.ConfidenceSummary{Claims: claims}
}

func TestCompareDeclaredToPersistedClassifiesEveryDeclaredClaim(t *testing.T) {
	summary := summaryOf(
		declaredClaim("C-1", "ranger", "assertion", 90),
		declaredClaim("C-2", "ranger", "assertion", 90),
		declaredClaim("C-3", "ranger", "assertion", 90),
		declaredClaim("C-4", "ranger", "assertion", 90),
		declaredClaim("C-5", "archivist", "assertion", 90),
	)
	records := []ConfidenceRecord{
		persistedRecord("C-1", "ranger", "assertion", 90, "high", ConfidenceCoverageReported),
		// C-2 is never recorded.
		persistedRecord("C-3", "ranger", "assertion", 90, "high", ConfidenceCoverageRejected),
		persistedRecord("C-4", "ranger", "assertion", 70, "medium", ConfidenceCoverageReported),
		// C-5 was recorded under another agent.
		persistedRecord("C-5", "handoff_challenge", "assertion", 90, "high", ConfidenceCoverageReported),
	}

	got := CompareDeclaredToPersisted(summary, records)

	assert.Equal(t, 5, got.DeclaredClaims)
	assert.Equal(t, 5, got.DeclaredAssertions)
	assert.Equal(t, 1, got.PersistedClaims)
	assert.Equal(t, []string{"ranger/C-2"}, got.Unpersisted)
	assert.Equal(t, []string{"ranger/C-3"}, got.PersistedRejected)
	assert.Equal(t, []string{"ranger/C-4"}, got.Mismatched)
	assert.Equal(t, []string{"archivist/C-5"}, got.UnderOtherAgent)
	assert.True(t, got.ReviewRecommended())
}

func TestCompareDeclaredToPersistedAllPersistedIsClean(t *testing.T) {
	summary := domain.ConfidenceSummary{
		Claims:        []domain.ConfidenceClaim{declaredClaim("C-1", "ranger", "assertion", 90)},
		OpenQuestions: []domain.ConfidenceClaim{declaredClaim("Q-1", "ranger", "question", 30)},
	}
	records := []ConfidenceRecord{
		persistedRecord("C-1", "ranger", "assertion", 90, "high", ConfidenceCoverageReported),
		persistedRecord("Q-1", "ranger", "question", 30, "low", ConfidenceCoverageReported),
	}

	got := CompareDeclaredToPersisted(summary, records)

	assert.Equal(t, 2, got.DeclaredClaims)
	assert.Equal(t, 1, got.DeclaredAssertions)
	assert.Equal(t, 1, got.DeclaredQuestions)
	assert.Equal(t, 2, got.PersistedClaims)
	assert.Equal(t, 1, got.PersistedQuestions, "a declared question persisted as a question is preserved")
	assert.Empty(t, got.Unpersisted)
	assert.False(t, got.ReviewRecommended())
}

func TestCompareDeclaredToPersistedMismatchRules(t *testing.T) {
	for name, change := range map[string]func(*ConfidenceRecord){
		"kind":            func(r *ConfidenceRecord) { r.ClaimKind = "question" },
		"percent":         func(r *ConfidenceRecord) { r.ConfidencePercent = 91 },
		"level":           func(r *ConfidenceRecord) { r.ConfidenceLevel = "medium" },
		"correlation_key": func(r *ConfidenceRecord) { r.CorrelationKey = "other" },
	} {
		t.Run(name, func(t *testing.T) {
			record := persistedRecord("C-1", "ranger", "assertion", 90, "high", ConfidenceCoverageReported)
			change(&record)

			got := CompareDeclaredToPersisted(summaryOf(declaredClaim("C-1", "ranger", "assertion", 90)), []ConfidenceRecord{record})

			assert.Equal(t, []string{"ranger/C-1"}, got.Mismatched)
			assert.Zero(t, got.PersistedClaims)
		})
	}
}

func TestCompareDeclaredToPersistedDerivesTheLevelWhenTheClaimOmitsIt(t *testing.T) {
	claim := declaredClaim("C-1", "ranger", "assertion", 70)
	record := persistedRecord("C-1", "ranger", "assertion", 70, "medium", ConfidenceCoverageReported)

	got := CompareDeclaredToPersisted(summaryOf(claim), []ConfidenceRecord{record})

	assert.Equal(t, 1, got.PersistedClaims)
}

func TestCompareDeclaredToPersistedIgnoresMissingRecordsAndCountsDuplicatesOnce(t *testing.T) {
	summary := summaryOf(declaredClaim("C-1", "ranger", "assertion", 90), declaredClaim("C-2", "ranger", "assertion", 90))
	records := []ConfidenceRecord{
		persistedRecord("C-1", "ranger", "assertion", 90, "high", ConfidenceCoverageReported),
		persistedRecord("C-1", "ranger", "assertion", 90, "high", ConfidenceCoverageReported),
		persistedRecord("C-2", "ranger", "assertion", 90, "high", ConfidenceCoverageMissing),
	}

	got := CompareDeclaredToPersisted(summary, records)

	assert.Equal(t, 1, got.PersistedClaims, "the duplicate line counts once")
	assert.Equal(t, []string{"ranger/C-2"}, got.Unpersisted, "a missing record never counts as persisted")
}

func TestCompareDeclaredToPersistedRejectedThenReportedIsPersisted(t *testing.T) {
	summary := summaryOf(declaredClaim("C-1", "ranger", "assertion", 90))
	records := []ConfidenceRecord{
		persistedRecord("C-1", "ranger", "assertion", 90, "high", ConfidenceCoverageRejected),
		persistedRecord("C-1", "ranger", "assertion", 90, "high", ConfidenceCoverageReported),
	}

	got := CompareDeclaredToPersisted(summary, records)

	assert.Equal(t, 1, got.PersistedClaims)
	assert.Empty(t, got.PersistedRejected)
}

func TestCompareDeclaredToPersistedSummaryWithMissingRecordHasNoBuckets(t *testing.T) {
	got := CompareDeclaredToPersisted(domain.ConfidenceSummary{MissingRecord: true}, nil)

	assert.True(t, got.MissingRecord)
	assert.Zero(t, got.DeclaredClaims)
	assert.False(t, got.ReviewRecommended())
}

func TestCompareDeclaredToPersistedEmptySummary(t *testing.T) {
	got := CompareDeclaredToPersisted(domain.ConfidenceSummary{}, []ConfidenceRecord{persistedRecord("C-1", "ranger", "assertion", 90, "high", ConfidenceCoverageReported)})

	require.Zero(t, got.DeclaredClaims)
	assert.False(t, got.ReviewRecommended())
}

func TestCompareDeclaredToPersistedIdsAreSorted(t *testing.T) {
	summary := summaryOf(declaredClaim("C-2", "ranger", "assertion", 90), declaredClaim("C-1", "ranger", "assertion", 90), declaredClaim("A-1", "archivist", "assertion", 90))

	got := CompareDeclaredToPersisted(summary, nil)

	assert.Equal(t, []string{"archivist/A-1", "ranger/C-1", "ranger/C-2"}, got.Unpersisted)
}

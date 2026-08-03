package treasure

import (
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// --- T1: field validation + round-trip ---

func TestValidateJewelEvidenceFields_Valid(t *testing.T) {
	t.Parallel()
	j := baseJewel("decision")
	j.EvidenceClass = domain.EvidenceClassExplicit
	j.EvidenceConfidence = domain.ConfidenceHigh
	j.ValidUntil = "2026-12-31T00:00:00Z"
	assert.NoError(t, ValidateJewelEntry(j, nil))
}

func TestValidateJewelEntry_ExistingJewelWithoutEvidenceFieldsStillValid(t *testing.T) {
	t.Parallel()
	// Additive-field round-trip: a pre-existing jewel with none of the new
	// fields set must keep validating unchanged.
	assert.NoError(t, ValidateJewelEntry(baseJewel("decision"), nil))
}

func TestValidateJewelEvidenceFields_InvalidValues(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		mutate  func(j Jewel) Jewel
		wantErr string
	}{
		{"invalid class", func(j Jewel) Jewel { j.EvidenceClass = "bogus"; return j }, "evidence_class"},
		{"invalid confidence", func(j Jewel) Jewel { j.EvidenceConfidence = "bogus"; return j }, "evidence_confidence"},
		{"invalid valid_until", func(j Jewel) Jewel { j.ValidUntil = "not-a-date"; return j }, "valid_until"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			j := tc.mutate(baseJewel("decision"))
			err := ValidateJewelEntry(j, nil)
			require.Error(t, err)
			assert.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestJewel_EvidenceFieldsYAMLRoundTrip(t *testing.T) {
	t.Parallel()
	j := baseJewel("decision")
	j.EvidenceClass = domain.EvidenceClassCorroboratedInference
	j.EvidenceConfidence = domain.ConfidenceMedium
	j.ValidUntil = "2026-12-31T00:00:00Z"

	out, err := yaml.Marshal(j)
	require.NoError(t, err)

	var roundTripped Jewel
	require.NoError(t, yaml.Unmarshal(out, &roundTripped))

	assert.Equal(t, j.EvidenceClass, roundTripped.EvidenceClass)
	assert.Equal(t, j.EvidenceConfidence, roundTripped.EvidenceConfidence)
	assert.Equal(t, j.ValidUntil, roundTripped.ValidUntil)
}

func TestJewel_YAMLWithoutEvidenceFieldsOmitsThem(t *testing.T) {
	t.Parallel()
	out, err := yaml.Marshal(baseJewel("decision"))
	require.NoError(t, err)

	assert.NotContains(t, string(out), "evidence_class:")
	assert.NotContains(t, string(out), "evidence_confidence:")
	assert.NotContains(t, string(out), "valid_until:")
}

// --- T3: expiration ---

func TestScanExpiredJewels_FlagsPastValidUntil(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	jewels := []Jewel{
		{ID: "jewel-1", ValidUntil: "2026-01-01T00:00:00Z"}, // expired
		{ID: "jewel-2", ValidUntil: "2027-01-01T00:00:00Z"}, // not expired
		{ID: "jewel-3"}, // no valid_until
	}
	found := ScanExpiredJewels("chest-a", jewels, now)
	require.Len(t, found, 1)
	assert.Equal(t, "jewel-1", found[0].JewelID)
	assert.Equal(t, "chest-a", found[0].ChestID)
}

func TestScanExpiredJewels_MalformedValidUntilNeverFlagged(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	jewels := []Jewel{{ID: "jewel-1", ValidUntil: "not-a-date"}}
	assert.Empty(t, ScanExpiredJewels("chest-a", jewels, now))
}

// --- T4: dedup ---

func TestScanDuplicateJewels_FlagsNormalizedMatch(t *testing.T) {
	t.Parallel()
	jewels := []Jewel{
		{ID: "jewel-1", Statement: "  The build uses Go 1.26  "},
		{ID: "jewel-2", Statement: "the build uses go 1.26"},
		{ID: "jewel-3", Statement: "Unrelated statement"},
	}
	found := ScanDuplicateJewels("chest-a", jewels)
	require.Len(t, found, 1)
	assert.Equal(t, "jewel-1", found[0].JewelIDA)
	assert.Equal(t, "jewel-2", found[0].JewelIDB)
}

func TestScanDuplicateJewels_EmptyStatementsNeverFlagged(t *testing.T) {
	t.Parallel()
	jewels := []Jewel{{ID: "jewel-1"}, {ID: "jewel-2"}}
	assert.Empty(t, ScanDuplicateJewels("chest-a", jewels))
}

func TestScanDuplicateJewels_NoSemanticMatch(t *testing.T) {
	t.Parallel()
	jewels := []Jewel{
		{ID: "jewel-1", Statement: "The build uses Go"},
		{ID: "jewel-2", Statement: "The build relies on Go"},
	}
	assert.Empty(t, ScanDuplicateJewels("chest-a", jewels))
}

// --- T5: conflict ---

func TestScanConflictingJewels_FlagsOverlappingRefsWithDifferingStatement(t *testing.T) {
	t.Parallel()
	jewels := []Jewel{
		{ID: "jewel-1", SourceRefs: []string{"chest#a"}, Statement: "X is true", Status: domain.JewelStatusAccepted},
		{ID: "jewel-2", SourceRefs: []string{"chest#a"}, Statement: "X is false", Status: domain.JewelStatusAccepted},
	}
	found := ScanConflictingJewels("chest-a", jewels)
	require.Len(t, found, 1)
	assert.Contains(t, found[0].Reason, "differing statement")
}

func TestScanConflictingJewels_FlagsOverlappingRefsWithDifferingStatus(t *testing.T) {
	t.Parallel()
	jewels := []Jewel{
		{ID: "jewel-1", SourceRefs: []string{"chest#a"}, Statement: "X is true", Status: domain.JewelStatusAccepted},
		{ID: "jewel-2", SourceRefs: []string{"chest#a"}, Statement: "X is true", Status: domain.JewelStatusDeprecated},
	}
	found := ScanConflictingJewels("chest-a", jewels)
	require.Len(t, found, 1)
	assert.Contains(t, found[0].Reason, "differing status")
}

func TestScanConflictingJewels_NoOverlapNeverFlagged(t *testing.T) {
	t.Parallel()
	jewels := []Jewel{
		{ID: "jewel-1", SourceRefs: []string{"chest#a"}, Statement: "X is true"},
		{ID: "jewel-2", SourceRefs: []string{"chest#b"}, Statement: "X is false"},
	}
	assert.Empty(t, ScanConflictingJewels("chest-a", jewels))
}

func TestScanConflictingJewels_OverlapWithIdenticalContentNeverFlagged(t *testing.T) {
	t.Parallel()
	jewels := []Jewel{
		{ID: "jewel-1", SourceRefs: []string{"chest#a"}, Statement: "X is true", Status: domain.JewelStatusAccepted},
		{ID: "jewel-2", SourceRefs: []string{"chest#a"}, Statement: "X is true", Status: domain.JewelStatusAccepted},
	}
	assert.Empty(t, ScanConflictingJewels("chest-a", jewels))
}

// --- combined report ---

func TestCheckEvidenceQuality_AggregatesAllThreeChecks(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	jewels := []Jewel{
		{ID: "jewel-1", ValidUntil: "2020-01-01T00:00:00Z"},
		{ID: "jewel-2", Statement: "dup"}, {ID: "jewel-3", Statement: "dup"},
		{ID: "jewel-4", SourceRefs: []string{"c#a"}, Statement: "X"},
		{ID: "jewel-5", SourceRefs: []string{"c#a"}, Statement: "Y"},
	}
	report := CheckEvidenceQuality("chest-a", jewels, now)
	assert.True(t, report.HasFindings())
	assert.Len(t, report.Expired, 1)
	assert.Len(t, report.Duplicates, 1)
	assert.Len(t, report.Conflicts, 1)
}

func TestCheckEvidenceQuality_EmptyJewelsHasNoFindings(t *testing.T) {
	t.Parallel()
	report := CheckEvidenceQuality("chest-a", nil, time.Now())
	assert.False(t, report.HasFindings())
}

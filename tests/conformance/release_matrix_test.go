package conformance_test

import (
	"embed"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/conformance"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/taxonomy-release-matrix.yaml
var releaseMatrixFixture embed.FS

func TestTaxonomyReleaseMatrixCoversEveryCandidateDimension(t *testing.T) {
	data, err := releaseMatrixFixture.ReadFile("testdata/taxonomy-release-matrix.yaml")
	require.NoError(t, err)
	matrix, err := conformance.DecodeMatrix(data)
	require.NoError(t, err)

	available := map[string]bool{"taxonomy-release-candidate": true}
	report, err := matrix.Evaluate(available)
	require.NoError(t, err)
	require.Equal(t, 16, report.RowCount)
	require.Len(t, report.Results, report.RowCount)
	for _, result := range report.Results {
		require.True(t, result.Passed, result.RowID)
	}

	wanted := map[string]struct{}{
		"fresh-install": {}, "supported-upgrade": {}, "supported-downgrade": {}, "rollback": {},
		"wizard": {}, "headless": {}, "offline-embedded": {}, "external-weapon": {},
		"missing-dependency": {}, "full-ranked": {}, "full-custom": {}, "short-ranked": {},
		"short-custom": {}, "roster-ranked": {}, "roster-custom": {}, "windows-compile": {},
	}
	for _, result := range report.Results {
		delete(wanted, result.RowID)
	}
	require.Empty(t, wanted)
}

func TestTaxonomyReleaseMatrixKeepsMissingDependencyFailClosed(t *testing.T) {
	data, err := releaseMatrixFixture.ReadFile("testdata/taxonomy-release-matrix.yaml")
	require.NoError(t, err)
	matrix, err := conformance.DecodeMatrix(data)
	require.NoError(t, err)

	for _, row := range matrix.Rows {
		if row.ID == "missing-dependency" {
			require.Equal(t, conformance.StateBlocked, row.ExpectedState)
			require.Equal(t, "readiness-check", row.AuthorityOwner)
			return
		}
	}
	t.Fatal("missing-dependency row not found")
}

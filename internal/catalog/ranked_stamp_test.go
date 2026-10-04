package catalog_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/catalog"
	"github.com/stretchr/testify/require"
)

func TestFindRankedStampRejectsMalformedCatalog(t *testing.T) {
	t.Parallel()

	_, _, err := catalog.FindRankedStamp([]byte(":\n\t- x"), "provider")
	require.ErrorContains(t, err, "parse catalog")
}

func TestFindRankedStampDelegatesCatalogValidation(t *testing.T) {
	t.Parallel()

	stamp, found, err := catalog.FindRankedStamp([]byte("schema_version: strategist-plugin-catalog/v2\nproviders:\n  - id: provider\n    ranked: true\n"), "provider")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "provider", stamp.ID)
}

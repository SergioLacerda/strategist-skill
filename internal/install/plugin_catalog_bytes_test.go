package install

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCatalogBytes_InvalidYAML(t *testing.T) {
	t.Parallel()
	_, err := parseCatalogBytes([]byte("schema_version: [unterminated"))
	require.Error(t, err)
	require.ErrorContains(t, err, "plugin catalog")
}

func TestParseCatalogBytes_MissingSchemaVersion(t *testing.T) {
	t.Parallel()
	_, err := parseCatalogBytes([]byte("providers:\n  - id: sniper\n    risk_score: controlled\n"))
	require.Error(t, err)
	require.ErrorContains(t, err, "schema_version is required")
}

func TestParseCatalogBytes_NoProviders(t *testing.T) {
	t.Parallel()
	_, err := parseCatalogBytes([]byte("schema_version: v1\nproviders: []\n"))
	require.Error(t, err)
	require.ErrorContains(t, err, "providers must have at least one entry")
}

func TestParseCatalogBytes_ProviderMissingIDOrRiskScore(t *testing.T) {
	t.Parallel()
	_, err := parseCatalogBytes([]byte("schema_version: v1\nproviders:\n  - id: sniper\n"))
	require.Error(t, err)
	require.ErrorContains(t, err, "provider id and risk_score are required")
}

func TestParseCatalogBytes_Success(t *testing.T) {
	t.Parallel()
	catalog, err := parseCatalogBytes([]byte("schema_version: v1\nproviders:\n  - id: sniper\n    risk_score: controlled\n"))
	require.NoError(t, err)
	assert.Equal(t, "v1", catalog.SchemaVersion)
	require.Len(t, catalog.Providers, 1)
	assert.Equal(t, "sniper", catalog.Providers[0].ID)
}

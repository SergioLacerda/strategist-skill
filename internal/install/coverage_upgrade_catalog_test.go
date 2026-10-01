package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogRegistryDriftFailureModesAndRefresh(t *testing.T) {
	t.Parallel()
	embedded, err := defaultsExtractor{}.ReadFile(pluginCatalogPath)
	require.NoError(t, err)

	dir := t.TempDir()
	s := Service{Extractor: defaultsExtractor{}}
	_, err = s.catalogRegistryDrifted(dir)
	require.ErrorContains(t, err, "upgrade: read "+pluginCatalogPath)

	require.NoError(t, os.MkdirAll(filepath.Join(dir, filepath.Dir(pluginCatalogPath)), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, pluginCatalogPath), embedded, 0o644))
	_, err = Service{Extractor: fixedFileExtractor{}}.catalogRegistryDrifted(dir)
	require.ErrorContains(t, err, "upgrade: read embedded")

	drifted, err := s.catalogRegistryDrifted(dir)
	require.NoError(t, err)
	assert.False(t, drifted, "an identical catalog has no registry drift")

	customized := []UpgradePlanEntry{{Path: "other.yaml", State: domain.UpgradeCustomized}, {Path: pluginCatalogPath, State: domain.UpgradeCustomized}}
	assert.Equal(t, 1, customizedCatalogIndex(customized))
	require.NoError(t, s.refreshCatalogEntry(dir, customized))
	assert.Equal(t, domain.UpgradeCustomized, customized[1].State, "no drift keeps the customization")

	managed := []UpgradePlanEntry{{Path: pluginCatalogPath, State: domain.UpgradeManaged}}
	assert.Equal(t, -1, customizedCatalogIndex(managed))
	require.NoError(t, s.refreshCatalogEntry(dir, managed))

	require.ErrorContains(t, s.refreshCatalogEntry(t.TempDir(), customized), "upgrade: read")

	require.NoError(t, os.WriteFile(filepath.Join(dir, pluginCatalogPath), []byte("schema_version: x\nproviders: []\nroles: [{id: ghost}]\n"), 0o644))
	drifted, err = s.catalogRegistryDrifted(dir)
	require.NoError(t, err)
	assert.True(t, drifted)
	require.NoError(t, s.refreshCatalogEntry(dir, customized))
	assert.Equal(t, domain.UpgradeAutoUpgrade, customized[1].State, "a drifted registry is promoted to an automatic upgrade")
}

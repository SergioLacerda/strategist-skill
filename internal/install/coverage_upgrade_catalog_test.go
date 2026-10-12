package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/installtest"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogRegistryDriftFailureModesAndRefresh(t *testing.T) {
	t.Parallel()
	embedded, err := installtest.DefaultsExtractor{}.ReadFile(names.PluginCatalogPath)
	require.NoError(t, err)

	dir := t.TempDir()
	s := Service{Extractor: installtest.DefaultsExtractor{}}
	_, err = s.catalogRegistryDrifted(dir)
	require.ErrorContains(t, err, "upgrade: read "+names.PluginCatalogPath)

	require.NoError(t, os.MkdirAll(filepath.Join(dir, filepath.Dir(names.PluginCatalogPath)), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, names.PluginCatalogPath), embedded, 0o644))
	_, err = Service{Extractor: installtest.FixedFileExtractor{}}.catalogRegistryDrifted(dir)
	require.ErrorContains(t, err, "upgrade: read embedded")

	drifted, err := s.catalogRegistryDrifted(dir)
	require.NoError(t, err)
	assert.False(t, drifted, "an identical catalog has no registry drift")

	customized := []UpgradePlanEntry{{Path: "other.yaml", State: domain.UpgradeCustomized}, {Path: names.PluginCatalogPath, State: domain.UpgradeCustomized}}
	assert.Equal(t, 1, customizedCatalogIndex(customized))
	require.NoError(t, s.refreshCatalogEntry(dir, customized))
	assert.Equal(t, domain.UpgradeCustomized, customized[1].State, "no drift keeps the customization")

	managed := []UpgradePlanEntry{{Path: names.PluginCatalogPath, State: domain.UpgradeManaged}}
	assert.Equal(t, -1, customizedCatalogIndex(managed))
	require.NoError(t, s.refreshCatalogEntry(dir, managed))

	require.ErrorContains(t, s.refreshCatalogEntry(t.TempDir(), customized), "upgrade: read")

	require.NoError(t, os.WriteFile(filepath.Join(dir, names.PluginCatalogPath), []byte("schema_version: x\nproviders: []\nroles: [{id: ghost}]\n"), 0o644))
	drifted, err = s.catalogRegistryDrifted(dir)
	require.NoError(t, err)
	assert.True(t, drifted)
	require.NoError(t, s.refreshCatalogEntry(dir, customized))
	assert.Equal(t, domain.UpgradeAutoUpgrade, customized[1].State, "a drifted registry is promoted to an automatic upgrade")
}

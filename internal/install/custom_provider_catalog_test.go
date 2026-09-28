package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanPluginOnboardingResolvesExplicitCustomProviderFromLocalRuntime(t *testing.T) {
	workspace := t.TempDir()
	home := t.TempDir()
	defaults := defaultsExtractor{}
	files := fixedFileExtractor{}
	for _, path := range []string{pluginCatalogPath, roleSlotMapPath, "roles/ranger.yaml", "roles/archivist.yaml", "roles/sniper.yaml"} {
		data, err := defaults.ReadFile(path)
		require.NoError(t, err)
		files[path] = fixedFileEntry{data: data}
	}
	t.Chdir(workspace)
	testutil.SetHome(t, home)

	providerDir := filepath.Join(workspace, ".agents", installedProvidersDirName, "team-brainstorming")
	require.NoError(t, os.MkdirAll(providerDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(providerDir, "SKILL.md"), []byte("---\nname: team-brainstorming\nmetadata:\n  version: \"2.0.0\"\n---\nbody\n"), 0o644))

	catalog, err := loadPluginCatalog(files)
	require.NoError(t, err)
	plan, err := planPluginOnboardingWithModes(files, catalog, map[string]string{
		"discovery":  "team-brainstorming",
		"refinement": "openspec-propose",
		"execution":  "sniper",
	}, map[string]string{
		"discovery":  domain.SlotBindingModeCustom,
		"refinement": domain.SlotBindingModeCustom,
		"execution":  domain.SlotBindingModeCustom,
	})
	require.NoError(t, err)

	resolution, ok := plan.CustomProviders["team-brainstorming"]
	require.True(t, ok)
	assert.Equal(t, connectors.ProviderOriginWorkspaceAgents, resolution.Package.Origin)
	assert.Equal(t, providerDir, resolution.Package.SeedPath)
	assert.Equal(t, filepath.Join(providerDir, "SKILL.md"), resolution.Package.Entrypoint)
	assert.Equal(t, "team-brainstorming", resolution.Package.Package.ID)

	discovery := plan.RoleMigration.Entries[0]
	require.Empty(t, discovery.ResolutionError)
	assert.Equal(t, "team-brainstorming", discovery.Resolved.Provider.ID)
	assert.Equal(t, domain.ProviderSourceExternal, discovery.Resolved.Provider.Source)
}

package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// customResolution resolves a real workspace host package through the planner,
// the same way the wizard does, and returns it with the strategist directory.
func customResolution(t *testing.T) (strategistDir string, resolution customProviderResolution) {
	t.Helper()
	workspace := t.TempDir()
	defaults := defaultsExtractor{}
	files := fixedFileExtractor{}
	for _, path := range []string{pluginCatalogPath, roleSlotMapPath, "roles/ranger.yaml", "roles/archivist.yaml", "roles/sniper.yaml"} {
		data, err := defaults.ReadFile(path)
		require.NoError(t, err)
		files[path] = fixedFileEntry{data: data}
	}
	t.Chdir(workspace)
	testutil.SetHome(t, t.TempDir())
	providerDir := filepath.Join(workspace, ".agents", installedProvidersDirName, "team-brainstorming")
	require.NoError(t, os.MkdirAll(providerDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(providerDir, "SKILL.md"), []byte("---\nname: team-brainstorming\nmetadata:\n  version: \"2.0.0\"\n---\nbody\n"), 0o644))
	scaffoldHostPackage(t, providerDir, "ranger", "discovery")

	catalog, err := loadPluginCatalog(files)
	require.NoError(t, err)
	plan, err := planPluginOnboardingWithModes(files, catalog,
		map[string]string{"discovery": "team-brainstorming", "refinement": "openspec-propose", "execution": "sniper"},
		map[string]string{"discovery": domain.SlotBindingModeCustom, "refinement": domain.SlotBindingModeCustom, "execution": domain.SlotBindingModeCustom})
	require.NoError(t, err)
	resolution, ok := plan.CustomProviders["team-brainstorming"]
	require.True(t, ok)
	strategistDir = filepath.Join(workspace, ".strategist")
	require.NoError(t, os.MkdirAll(strategistDir, 0o755))
	return strategistDir, resolution
}

func bareCustomLock() domain.PluginLockFile {
	return domain.PluginLockFile{
		Inventory: domain.PluginInventory{Instances: []domain.InstalledInstance{
			{ID: "team-brainstorming"}, {ID: "other"},
		}},
		Bindings: []domain.SlotBinding{{Slot: "discovery", InstalledInstanceID: "team-brainstorming", Generation: 3}},
		Lock: domain.PluginLock{Nodes: []domain.PluginLockNode{
			{ID: "team-brainstorming"}, {ID: "role:team-brainstorming"}, {ID: "keep"},
		}},
	}
}

func TestMaterializeCustomProvidersWritesTheInstalledPackageAndBindsIt(t *testing.T) {
	dir, resolution := customResolution(t)
	providers := map[string]customProviderResolution{"team-brainstorming": resolution}

	lock, err := materializeCustomProviders(dir, bareCustomLock(), providers)
	require.NoError(t, err)

	instance := lock.Bindings[0].InstalledInstanceID
	assert.Contains(t, instance, "team-brainstorming@")
	assert.FileExists(t, filepath.Join(dir, "providers", instance, "package.yaml"))
	assert.FileExists(t, filepath.Join(dir, "providers", instance, "adapter.yaml"))
	assert.Equal(t, instance, lock.Inventory.Instances[0].ID, "the bare instance is renamed")
	assert.Equal(t, "other", lock.Inventory.Instances[1].ID)
	assert.Equal(t, resolution.Package.Origin, lock.Inventory.Instances[0].ProviderOrigin)
	assert.NotEmpty(t, lock.Lock.GraphDigest)
	assert.Equal(t, lock.Lock.GraphDigest, lock.Lock.ResolutionID)
	var ids []string
	for _, node := range lock.Lock.Nodes {
		ids = append(ids, node.ID)
	}
	assert.Contains(t, ids, "keep")
	assert.NotContains(t, ids, "role:team-brainstorming", "the bare-id role-binding node is replaced")
	assert.Len(t, ids, 4, "one kept node plus the three Custom nodes")

	annotated := bareCustomLock()
	annotateCustomProviderInstances(&annotated, providers)
	assert.Equal(t, "host", annotated.Inventory.Instances[0].ConnectorID)
	assert.Equal(t, resolution.Package.SeedPath, annotated.Inventory.Instances[0].SeedPath)
	assert.Empty(t, annotated.Inventory.Instances[1].ConnectorID)

	assert.Equal(t, []string{"a", "b"}, sortedProviderIDs(map[string]customProviderResolution{"b": {}, "a": {}}))
}

func TestMaterializeCustomProvidersRollsBackCreatedDirectoriesOnFailure(t *testing.T) {
	dir, resolution := customResolution(t)
	noBinding := bareCustomLock()
	noBinding.Bindings = nil

	_, err := materializeCustomProviders(dir, noBinding, map[string]customProviderResolution{"team-brainstorming": resolution})
	require.ErrorContains(t, err, `materialize custom provider "team-brainstorming"`)
	require.ErrorContains(t, err, "read the slot's current binding")
	instances, readErr := filepath.Glob(filepath.Join(dir, "providers", "*@*"))
	require.NoError(t, readErr)
	assert.Empty(t, instances, "the installed package directory is removed again")

	_, _, err = currentCustomBindingState(noBinding, "discovery")
	require.Error(t, err)
	generation, status, err := currentCustomBindingState(domain.PluginLockFile{Bindings: []domain.SlotBinding{{Slot: "discovery"}}}, "discovery")
	require.NoError(t, err)
	assert.Equal(t, int64(1), generation, "a missing generation floors at 1")
	assert.Equal(t, "active", status)
}

func TestStageCustomPackageRejectsAnInvalidNormalizedPackage(t *testing.T) {
	dir, resolution := customResolution(t)
	resolution.Package.Package.ID = ""
	_, _, err := stageCustomPackage(dir, resolution)
	require.ErrorContains(t, err, "normalized package contract is invalid")

	resolution.Package.Package.ID = "team-brainstorming"
	resolution.Declaration.RiskScore = "not-a-risk"
	_, _, adapterErr := normalizedCustomPackage(resolution)
	require.Error(t, adapterErr)

	assert.Equal(t, "x", firstNonEmpty("", "  ", "x"))
	assert.Empty(t, firstNonEmpty())
}

func TestCommitStagedDirectoryFailureModes(t *testing.T) {
	base := t.TempDir()
	blocker := filepath.Join(base, "providers")
	require.NoError(t, os.WriteFile(blocker, nil, 0o644))
	require.ErrorContains(t, commitStagedDirectory(filepath.Join(blocker, "inst"), nil), "create provider staging directory")
}

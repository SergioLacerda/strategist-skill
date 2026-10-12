package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/installtest"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshInstalledRankedBindingsBranches(t *testing.T) {
	t.Parallel()

	missing := t.TempDir()
	require.NoError(t, Service{Extractor: installtest.FixedFileExtractor{}}.refreshInstalledRankedBindings(missing), "no plugins.lock is a no-op")

	broken := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(broken, names.PluginLockFileName), []byte("bindings: [unclosed"), 0o644))
	require.ErrorContains(t, Service{Extractor: installtest.FixedFileExtractor{}}.refreshInstalledRankedBindings(broken), "read plugins.lock")

	custom := t.TempDir()
	require.NoError(t, plugincatalog.WritePluginLockFile(custom, domain.PluginLockFile{Bindings: []domain.SlotBinding{{Slot: "discovery", InstalledInstanceID: "x"}}}))
	require.NoError(t, Service{Extractor: installtest.FixedFileExtractor{}}.refreshInstalledRankedBindings(custom), "a Custom binding is left untouched")

	ranked := t.TempDir()
	require.NoError(t, plugincatalog.WritePluginLockFile(ranked, domain.PluginLockFile{Bindings: []domain.SlotBinding{{Slot: "discovery", InstalledInstanceID: "x", Mode: domain.SlotBindingModeRanked}}}))
	require.ErrorContains(t, Service{Extractor: installtest.FixedFileExtractor{}}.refreshInstalledRankedBindings(ranked), "load plugin catalog")
}

func TestLoadSilentBindingPlanErrors(t *testing.T) {
	t.Parallel()

	_, _, _, err := Service{Extractor: installtest.FixedFileExtractor{}}.loadSilentBindingPlan([]byte("slots: [unclosed"))
	require.ErrorContains(t, err, "parse active.yaml template")

	_, _, _, err = Service{Extractor: installtest.FixedFileExtractor{}}.loadSilentBindingPlan([]byte("slots: {}\n"))
	require.ErrorContains(t, err, "load plugin catalog")

	defaults := installtest.DefaultsExtractor{}
	files := installtest.FixedFileExtractor{}
	for _, path := range []string{names.PluginCatalogPath, names.RoleSlotMapPath, "roles/ranger.yaml", "roles/archivist.yaml", "roles/sniper.yaml"} {
		data, readErr := defaults.ReadFile(path)
		require.NoError(t, readErr)
		files[path] = installtest.FixedFileEntry{Data: data}
	}
	_, _, _, err = Service{Extractor: files}.loadSilentBindingPlan([]byte("slots:\n  discovery: no-such-provider\n  refinement: openspec-propose\n  execution: sniper\n"))
	require.ErrorContains(t, err, "plugin onboarding plan")
}

func TestActivateSilentRoleProviderBindingsRejectsAnInvalidTemplate(t *testing.T) {
	t.Parallel()
	err := Service{Extractor: installtest.FixedFileExtractor{}}.activateSilentRoleProviderBindings(t.TempDir(), []byte("slots: [unclosed"))
	require.ErrorContains(t, err, "parse active.yaml template")
}

func TestPersistSilentBindings(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, persistSilentBindings(dir, domain.PluginLockFile{}))
	assert.NoFileExists(t, filepath.Join(dir, names.PluginLockFileName), "an empty lock is not written")

	lock := domain.PluginLockFile{Bindings: []domain.SlotBinding{{Slot: "discovery", InstalledInstanceID: "x"}}}
	require.NoError(t, persistSilentBindings(dir, lock))
	assert.FileExists(t, filepath.Join(dir, names.PluginLockFileName))
	blocker := filepath.Join(dir, "blocker")
	require.NoError(t, os.WriteFile(blocker, nil, 0o644))
	require.ErrorContains(t, persistSilentBindings(blocker, lock), "write plugins.lock")
}

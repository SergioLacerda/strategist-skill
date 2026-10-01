package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshInstalledRankedBindingsBranches(t *testing.T) {
	t.Parallel()

	missing := t.TempDir()
	require.NoError(t, Service{Extractor: fixedFileExtractor{}}.refreshInstalledRankedBindings(missing), "no plugins.lock is a no-op")

	broken := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(broken, pluginLockFileName), []byte("bindings: [unclosed"), 0o644))
	require.ErrorContains(t, Service{Extractor: fixedFileExtractor{}}.refreshInstalledRankedBindings(broken), "read plugins.lock")

	custom := t.TempDir()
	require.NoError(t, writePluginLockFile(custom, domain.PluginLockFile{Bindings: []domain.SlotBinding{{Slot: "discovery", InstalledInstanceID: "x"}}}))
	require.NoError(t, Service{Extractor: fixedFileExtractor{}}.refreshInstalledRankedBindings(custom), "a Custom binding is left untouched")

	ranked := t.TempDir()
	require.NoError(t, writePluginLockFile(ranked, domain.PluginLockFile{Bindings: []domain.SlotBinding{{Slot: "discovery", InstalledInstanceID: "x", Mode: domain.SlotBindingModeRanked}}}))
	require.ErrorContains(t, Service{Extractor: fixedFileExtractor{}}.refreshInstalledRankedBindings(ranked), "load plugin catalog")
}

func TestLoadSilentBindingPlanErrors(t *testing.T) {
	t.Parallel()

	_, _, _, err := Service{Extractor: fixedFileExtractor{}}.loadSilentBindingPlan([]byte("slots: [unclosed"))
	require.ErrorContains(t, err, "parse active.yaml template")

	_, _, _, err = Service{Extractor: fixedFileExtractor{}}.loadSilentBindingPlan([]byte("slots: {}\n"))
	require.ErrorContains(t, err, "load plugin catalog")

	defaults := defaultsExtractor{}
	files := fixedFileExtractor{}
	for _, path := range []string{pluginCatalogPath, roleSlotMapPath, "roles/ranger.yaml", "roles/archivist.yaml", "roles/sniper.yaml"} {
		data, readErr := defaults.ReadFile(path)
		require.NoError(t, readErr)
		files[path] = fixedFileEntry{data: data}
	}
	_, _, _, err = Service{Extractor: files}.loadSilentBindingPlan([]byte("slots:\n  discovery: no-such-provider\n  refinement: openspec-propose\n  execution: sniper\n"))
	require.ErrorContains(t, err, "plugin onboarding plan")
}

func TestActivateSilentRoleProviderBindingsRejectsAnInvalidTemplate(t *testing.T) {
	t.Parallel()
	err := Service{Extractor: fixedFileExtractor{}}.activateSilentRoleProviderBindings(t.TempDir(), []byte("slots: [unclosed"))
	require.ErrorContains(t, err, "parse active.yaml template")
}

func TestPersistSilentBindings(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, persistSilentBindings(dir, domain.PluginLockFile{}))
	assert.NoFileExists(t, filepath.Join(dir, pluginLockFileName), "an empty lock is not written")

	lock := domain.PluginLockFile{Bindings: []domain.SlotBinding{{Slot: "discovery", InstalledInstanceID: "x"}}}
	require.NoError(t, persistSilentBindings(dir, lock))
	assert.FileExists(t, filepath.Join(dir, pluginLockFileName))
	blocker := filepath.Join(dir, "blocker")
	require.NoError(t, os.WriteFile(blocker, nil, 0o644))
	require.ErrorContains(t, persistSilentBindings(blocker, lock), "write plugins.lock")
}

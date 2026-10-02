package install

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadNormalizedSilentConfigFailures(t *testing.T) {
	t.Parallel()
	_, err := Service{Extractor: fixedFileExtractor{}}.readNormalizedSilentConfig()
	require.ErrorContains(t, err, "install: read template")

	bad := fixedFileExtractor{epicStandaloneTemplatePath: {data: []byte("language: [unclosed")}}
	_, err = Service{Extractor: bad}.readNormalizedSilentConfig()
	require.ErrorContains(t, err, "install: normalize language")
}

func TestProviderManifestAndSelectionFailures(t *testing.T) {
	t.Parallel()
	_, err := providerManifestBytes(fixedFileExtractor{}, "brainstorming")
	require.ErrorContains(t, err, "load plugin catalog for brainstorming")

	_, err = providerManifestBytes(defaultsExtractor{}, "definitely-not-a-provider")
	require.ErrorContains(t, err, "not found")

	err = Service{Extractor: fixedFileExtractor{}}.writeSelectedProviderManifest(t.TempDir(), "brainstorming")
	require.ErrorContains(t, err, "resolve installable providers for brainstorming")

	require.NoError(t, Service{Extractor: defaultsExtractor{}}.writeSelectedProviderManifest(t.TempDir(), "not-installable-by-default"))
	err = Service{Extractor: fixedFileExtractor{}}.writeSelectedProviderManifests(t.TempDir(), domain.WizardConfig{DiscoveryProvider: "brainstorming"})
	require.Error(t, err)
}

func TestPersistWizardConfigFailures(t *testing.T) {
	t.Parallel()
	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, nil, 0o644))
	wc := domain.WizardConfig{ResolvedPluginLock: domain.PluginLockFile{Bindings: []domain.SlotBinding{{Slot: "discovery", InstalledInstanceID: "x"}}}}
	err := Service{Extractor: defaultsExtractor{}}.persistWizardConfig(blocker, wc)
	require.ErrorContains(t, err, "install: write plugins.lock")

	err = Service{Extractor: fixedFileExtractor{}}.persistWizardConfig(t.TempDir(), domain.WizardConfig{DiscoveryProvider: "brainstorming"})
	require.ErrorContains(t, err, "install: write provider manifests")
}

func TestUpgradeFileOperationFailures(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s := Service{Extractor: fixedFileExtractor{"a.txt": {data: []byte("a")}, "bad.txt": {err: errors.New("boom")}}}

	require.ErrorContains(t, s.writeUpgradeFile(dir, "bad.txt"), "upgrade: read embedded bad.txt")
	require.Error(t, s.writeUpgradeFile(dir, "../escape.txt"))
	require.ErrorContains(t, s.writeUpgradeFiles(dir, []string{"bad.txt"}), "extract runtime tree")
	require.NoError(t, s.writeUpgradeFiles(dir, []string{"a.txt"}))
	assert.FileExists(t, filepath.Join(dir, "a.txt"))

	require.Error(t, s.snapshotUpgradePath(dir, t.TempDir(), "../escape.txt"))
	require.NoError(t, s.snapshotUpgradePath(dir, t.TempDir(), "absent.txt"), "a file that never existed has nothing to snapshot")
	require.NoError(t, os.Mkdir(filepath.Join(dir, "adir"), 0o755))
	require.ErrorContains(t, s.snapshotUpgradePath(dir, t.TempDir(), "adir"), "snapshot read")

	backup := t.TempDir()
	require.NoError(t, s.snapshotUpgradePath(dir, backup, "a.txt"))
	assert.FileExists(t, filepath.Join(backup, "a.txt"))
}

func TestReconcileUpgradedStateAndFinalizeFailures(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, pluginLockFileName), []byte("bindings: [unclosed"), 0o644))
	err := Service{Extractor: fixedFileExtractor{}}.reconcileUpgradedState(dir, UpgradePlan{LockMigration: []string{"discovery"}})
	require.ErrorContains(t, err, "upgrade: migrate plugins.lock")

	err = Service{Extractor: fixedFileExtractor{}}.reconcileUpgradedState(dir, UpgradePlan{})
	require.ErrorContains(t, err, "upgrade: reconcile ranked runtimes")

	s := Service{Extractor: fixedFileExtractor{"bad.txt": {err: errors.New("boom")}}}
	require.ErrorContains(t, s.finalizeUpgrade(t.TempDir(), UpgradePlan{}, []string{"bad.txt"}, nil), "upgrade: read embedded")
}

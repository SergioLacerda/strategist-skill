package install

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/installtest"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadNormalizedSilentConfigFailures(t *testing.T) {
	t.Parallel()
	_, err := Service{Extractor: installtest.FixedFileExtractor{}}.readNormalizedSilentConfig()
	require.ErrorContains(t, err, "install: read template")

	bad := installtest.FixedFileExtractor{names.EpicStandaloneTemplatePath: {Data: []byte("language: [unclosed")}}
	_, err = Service{Extractor: bad}.readNormalizedSilentConfig()
	require.ErrorContains(t, err, "install: normalize language")
}

func TestPersistWizardConfigFailures(t *testing.T) {
	t.Parallel()
	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, nil, 0o644))
	wc := domain.WizardConfig{ResolvedPluginLock: domain.PluginLockFile{Bindings: []domain.SlotBinding{{Slot: "discovery", InstalledInstanceID: "x"}}}}
	err := Service{Extractor: installtest.DefaultsExtractor{}}.persistWizardConfig(blocker, wc)
	require.ErrorContains(t, err, "install: write plugins.lock")

}

func TestUpgradeFileOperationFailures(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s := Service{Extractor: installtest.FixedFileExtractor{"a.txt": {Data: []byte("a")}, "bad.txt": {Err: errors.New("boom")}}}

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
	require.NoError(t, os.WriteFile(filepath.Join(dir, names.PluginLockFileName), []byte("bindings: [unclosed"), 0o644))
	err := Service{Extractor: installtest.FixedFileExtractor{}}.reconcileUpgradedState(dir, UpgradePlan{LockMigration: []string{"discovery"}})
	require.ErrorContains(t, err, "upgrade: migrate plugins.lock")

	err = Service{Extractor: installtest.FixedFileExtractor{}}.reconcileUpgradedState(dir, UpgradePlan{})
	require.ErrorContains(t, err, "upgrade: reconcile ranked runtimes")

	s := Service{Extractor: installtest.FixedFileExtractor{"bad.txt": {Err: errors.New("boom")}}}
	require.ErrorContains(t, s.finalizeUpgrade(t.TempDir(), UpgradePlan{}, []string{"bad.txt"}, nil), "upgrade: read embedded")
}

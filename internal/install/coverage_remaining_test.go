package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/installtest"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplySilentConfigReportsBackupFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte("mode: epic\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".upgrade-backups"), []byte("not a directory"), 0o644))

	err := (Service{Extractor: installtest.MinimalExtractor{}}).applySilentConfig(
		context.Background(), dir, domain.InstallConfig{ResetConfig: true},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "back up configuration")
}

func TestApplySilentConfigReportsActiveYAMLWriteFailure(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "strategist-file")
	require.NoError(t, os.WriteFile(path, []byte("not a directory"), 0o644))

	err := (Service{Extractor: installtest.MinimalExtractor{}}).applySilentConfig(
		context.Background(), path, domain.InstallConfig{},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "install:")
}

func TestRunShimStepCanBeDisabled(t *testing.T) {
	t.Parallel()
	paths, err := (Service{}).runShimStep(context.Background(), domain.InstallConfig{NoShim: true})
	require.NoError(t, err)
	assert.Nil(t, paths)
}

func TestEnsureProjectGitignoreReportsWriteFailure(t *testing.T) {
	t.Parallel()
	target := filepath.Join(t.TempDir(), "target-file")
	require.NoError(t, os.WriteFile(target, []byte("not a directory"), 0o644))

	_, err := ensureProjectGitignore(domain.InstallConfig{Target: target})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "install: gitignore")
}

func TestRefreshRankedLockReportsMissingProvider(t *testing.T) {
	t.Parallel()
	lock := domain.PluginLockFile{Bindings: []domain.SlotBinding{
		{Slot: "discovery", InstalledInstanceID: "missing", Mode: domain.SlotBindingModeRanked},
	}}

	_, err := refreshRankedLock(plugincatalog.PluginCatalog{}, lock)
	require.Error(t, err)
}

func TestApplyWizardConfigReportsBackupFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte("mode: epic\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".upgrade-backups"), []byte("not a directory"), 0o644))
	writeWizardLevelingFixture(t, dir)

	svc := Service{
		Extractor:      installtest.MinimalExtractor{},
		WizardPrompter: p("en\nen\nen\nen\nepic\n.analysis\n\n\nbrainstorming\nopenspec-explore\nfixture-provider\n\n"),
	}
	err := svc.applyWizardConfig(context.Background(), dir, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "back up configuration")
}

func TestRunShimStepReportsShimFailure(t *testing.T) {
	t.Parallel()
	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("not a directory"), 0o644))

	_, err := (Service{Extractor: installtest.MinimalExtractor{}}).runShimStep(
		context.Background(), domain.InstallConfig{Target: t.TempDir(), ShimPath: filepath.Join(blocker, "SKILL.md")},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "shim")
}

func TestRunInstallStepsReportsRuntimePreparationFailure(t *testing.T) {
	t.Parallel()
	tx := newInstallTransaction(filepath.Join(t.TempDir(), ".strategist"))
	_, err := (Service{Extractor: failingInstallExtractor{}, Compiler: nopCompiler{}}).runInstallSteps(
		context.Background(), tx.strategistDir, domain.InstallConfig{}, runtimeDefaultPlan{}, tx,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "extract")
}

func TestRefreshInstalledRankedBindingsReportsRefreshFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lock := domain.PluginLockFile{Bindings: []domain.SlotBinding{
		{Slot: "discovery", InstalledInstanceID: "missing", Mode: domain.SlotBindingModeRanked},
	}}
	require.NoError(t, plugincatalog.WritePluginLockFile(dir, lock))

	err := (Service{Extractor: installtest.DefaultsExtractor{}}).refreshInstalledRankedBindings(dir)
	require.Error(t, err)
}

func TestActivateSilentRoleProviderBindingsReportsPersistenceFailure(t *testing.T) {
	t.Parallel()
	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("not a directory"), 0o644))

	err := (Service{Extractor: slottedTemplateExtractor{}}).activateSilentRoleProviderBindings(
		blocker, []byte("mode: epic\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n  refinement: openspec-propose\n  execution: sniper\n"),
	)
	require.Error(t, err)
}

func TestRuntimeDefaultHelpersReportFilesystemFailures(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "runtime-file"), 0o755))
	_, err := planRuntimeDefaultFile(
		dir, "runtime-file", "embedded", domain.InstallManifest{}, false, runtimeDefaultPolicy{},
	)
	require.Error(t, err)

	file := filepath.Join(t.TempDir(), "strategist-file")
	require.NoError(t, os.WriteFile(file, []byte("not a directory"), 0o644))
	err = (Service{Extractor: installtest.FixedFileExtractor{"runtime-file": {Data: []byte("runtime")}}}).applyRuntimeDefaultFile(
		file, "runtime-file", domain.RuntimeDecisionWriteMissing,
	)
	require.Error(t, err)
}

func TestFinalizeInstallReportsManifestFailure(t *testing.T) {
	t.Parallel()
	file := filepath.Join(t.TempDir(), "strategist-file")
	require.NoError(t, os.WriteFile(file, []byte("not a directory"), 0o644))

	err := (Service{Compiler: nopCompiler{}}).finalizeInstall(
		context.Background(), domain.InstallConfig{Target: t.TempDir()}, file, runtimeDefaultPlan{}, nil,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "manifest")
}

func TestCatalogRegistryDriftedReportsWorkspaceAndEmbeddedReadFailures(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, err := (Service{Extractor: installtest.FixedFileExtractor{}}).catalogRegistryDrifted(dir)
	require.Error(t, err)

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "plugins"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugins", "catalog.yaml"), []byte("catalog"), 0o644))
	_, err = (Service{Extractor: installtest.FixedFileExtractor{}}).catalogRegistryDrifted(dir)
	require.Error(t, err)
}

type failingInstallExtractor struct{}

func (failingInstallExtractor) Extract(string, bool) error { return errors.New("extract failed") }

func (failingInstallExtractor) ReadFile(string) ([]byte, error) {
	return nil, errors.New("read failed")
}

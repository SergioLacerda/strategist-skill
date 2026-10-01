package install

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryRootResolution(t *testing.T) {
	t.Parallel()
	root := repoRoot()
	require.NotEmpty(t, root)
	assert.True(t, repositoryRoot(root))
	assert.Equal(t, root, repositoryRootFrom(filepath.Join(root, "internal", "install")))
	assert.False(t, repositoryRoot(t.TempDir()))
	assert.Empty(t, repositoryRootFrom(t.TempDir()))
}

func TestConformanceDigests(t *testing.T) {
	t.Parallel()
	for _, role := range []string{"ranger", "archivist", "sniper"} {
		digest, err := testSuiteDigest(role)
		require.NoError(t, err, role)
		assert.Contains(t, digest, "sha256:")
	}
	_, err := testSuiteDigest("scout")
	require.ErrorContains(t, err, "no conformance test file pinned")

	connector, err := connectorDigest()
	require.NoError(t, err)
	policy, err := policyDigest()
	require.NoError(t, err)
	assert.NotEqual(t, connector, policy)

	_, err = hostAPIContractDigest(t.TempDir(), "ranger")
	require.ErrorContains(t, err, "digest ")
	_, err = digestFiles(filepath.Join(t.TempDir(), "absent"))
	require.Error(t, err)
}

func TestLoadRankedBindingsBranches(t *testing.T) {
	t.Parallel()
	missing, err := loadRankedBindings(t.TempDir())
	require.NoError(t, err)
	assert.Nil(t, missing)

	asDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(asDir, "plugins.lock"), 0o755))
	_, err = loadRankedBindings(asDir)
	require.ErrorContains(t, err, "read plugins.lock for ranked runtime")

	broken := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(broken, "plugins.lock"), []byte("bindings: [unclosed"), 0o644))
	_, err = loadRankedBindings(broken)
	require.ErrorContains(t, err, "parse plugins.lock for ranked runtime")

	mixed := t.TempDir()
	require.NoError(t, writePluginLockFile(mixed, domain.PluginLockFile{Bindings: []domain.SlotBinding{
		{Slot: "discovery", InstalledInstanceID: "a"},
		{Slot: "refinement", InstalledInstanceID: "b", Mode: domain.SlotBindingModeRanked},
	}}))
	ranked, err := loadRankedBindings(mixed)
	require.NoError(t, err)
	require.Len(t, ranked, 1)
	assert.Equal(t, "refinement", ranked[0].Slot)

	require.NoError(t, prepareRankedProviderRuntimes(context.Background(), t.TempDir()), "no ranked bindings is a no-op")
	require.ErrorContains(t, prepareRankedProviderRuntimes(context.Background(), mixed), "ranked runtime")
}

func TestLoadRankedRuntimeInputsFailureModes(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, _, err := loadRankedRuntimeInputs(dir)
	require.ErrorContains(t, err, "read role map")

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "roles", "default.yaml"), []byte("[unclosed"), 0o644))
	_, _, err = loadRankedRuntimeInputs(dir)
	require.ErrorContains(t, err, "parse role map")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "roles", "default.yaml"), []byte("discovery: ranger\n"), 0o644))
	_, _, err = loadRankedRuntimeInputs(dir)
	require.ErrorContains(t, err, "read catalog")

	require.NoError(t, os.MkdirAll(filepath.Join(dir, filepath.Dir(pluginCatalogPath)), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, pluginCatalogPath), []byte("providers: [unclosed"), 0o644))
	_, _, err = loadRankedRuntimeInputs(dir)
	require.ErrorContains(t, err, "parse catalog")

	require.NoError(t, os.WriteFile(filepath.Join(dir, pluginCatalogPath), []byte("schema_version: strategist-plugin-catalog/v2\nproviders:\n  - id: sniper\n    risk_score: controlled\n    compatibility_source: native_role\n"), 0o644))
	roles, _, err := loadRankedRuntimeInputs(dir)
	require.NoError(t, err)
	assert.Equal(t, "ranger", roles["discovery"])
}

func TestCommitStagedDirectoryWriteAndReplaceFailures(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	require.ErrorContains(t, commitStagedDirectory(filepath.Join(base, "inst"), map[string][]byte{"no/such/dir.yaml": []byte("x")}), "write staged")
}

package embeddedweapon_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/customprovider"
	"github.com/SergioLacerda/strategist-skill/internal/install/embeddedweapon"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/SergioLacerda/strategist-skill/internal/install/rankedbinding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryRootResolution(t *testing.T) {
	t.Parallel()
	root := embeddedweapon.RepoRoot()
	require.NotEmpty(t, root)
	assert.True(t, embeddedweapon.RepositoryRoot(root))
	assert.Equal(t, root, embeddedweapon.RepositoryRootFrom(filepath.Join(root, "internal", "install")))
	assert.False(t, embeddedweapon.RepositoryRoot(t.TempDir()))
	assert.Empty(t, embeddedweapon.RepositoryRootFrom(t.TempDir()))
}

func TestConformanceDigests(t *testing.T) {
	t.Parallel()
	for _, role := range []string{"ranger", "archivist", "sniper"} {
		digest, err := embeddedweapon.TestSuiteDigest(role)
		require.NoError(t, err, role)
		assert.Contains(t, digest, "sha256:")
	}
	_, err := embeddedweapon.TestSuiteDigest("scout")
	require.ErrorContains(t, err, "no conformance test file pinned")

	connector, err := embeddedweapon.ConnectorDigest()
	require.NoError(t, err)
	policy, err := embeddedweapon.PolicyDigest()
	require.NoError(t, err)
	assert.NotEqual(t, connector, policy)

	_, err = embeddedweapon.HostAPIContractDigest(t.TempDir(), "ranger")
	require.ErrorContains(t, err, "digest ")
	_, err = embeddedweapon.DigestFiles(filepath.Join(t.TempDir(), "absent"))
	require.Error(t, err)
}

func TestLoadRankedBindingsBranches(t *testing.T) {
	t.Parallel()
	missing, err := rankedbinding.LoadRankedBindings(t.TempDir())
	require.NoError(t, err)
	assert.Nil(t, missing)

	asDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(asDir, "plugins.lock"), 0o755))
	_, err = rankedbinding.LoadRankedBindings(asDir)
	require.ErrorContains(t, err, "read plugins.lock for ranked runtime")

	broken := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(broken, "plugins.lock"), []byte("bindings: [unclosed"), 0o644))
	_, err = rankedbinding.LoadRankedBindings(broken)
	require.ErrorContains(t, err, "parse plugins.lock for ranked runtime")

	mixed := t.TempDir()
	require.NoError(t, plugincatalog.WritePluginLockFile(mixed, domain.PluginLockFile{Bindings: []domain.SlotBinding{
		{Slot: "discovery", InstalledInstanceID: "a"},
		{Slot: "refinement", InstalledInstanceID: "b", Mode: domain.SlotBindingModeRanked},
	}}))
	ranked, err := rankedbinding.LoadRankedBindings(mixed)
	require.NoError(t, err)
	require.Len(t, ranked, 1)
	assert.Equal(t, "refinement", ranked[0].Slot)

	require.NoError(t, rankedbinding.PrepareRankedProviderRuntimes(context.Background(), t.TempDir()), "no ranked bindings is a no-op")
	require.ErrorContains(t, rankedbinding.PrepareRankedProviderRuntimes(context.Background(), mixed), "ranked runtime")
}

func TestLoadRankedRuntimeInputsFailureModes(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, _, err := rankedbinding.LoadRankedRuntimeInputs(dir)
	require.ErrorContains(t, err, "read role map")

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "roles", "default.yaml"), []byte("[unclosed"), 0o644))
	_, _, err = rankedbinding.LoadRankedRuntimeInputs(dir)
	require.ErrorContains(t, err, "parse role map")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "roles", "default.yaml"), []byte("discovery: ranger\n"), 0o644))
	_, _, err = rankedbinding.LoadRankedRuntimeInputs(dir)
	require.ErrorContains(t, err, "read catalog")

	require.NoError(t, os.MkdirAll(filepath.Join(dir, filepath.Dir(names.PluginCatalogPath)), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, names.PluginCatalogPath), []byte("providers: [unclosed"), 0o644))
	_, _, err = rankedbinding.LoadRankedRuntimeInputs(dir)
	require.ErrorContains(t, err, "parse catalog")

	require.NoError(t, os.WriteFile(filepath.Join(dir, names.PluginCatalogPath), []byte("schema_version: strategist-plugin-catalog/v2\nproviders:\n  - id: sniper\n    risk_score: controlled\n    compatibility_source: native_role\n"), 0o644))
	roles, _, err := rankedbinding.LoadRankedRuntimeInputs(dir)
	require.NoError(t, err)
	assert.Equal(t, "ranger", roles["discovery"])
}

func TestCommitStagedDirectoryWriteAndReplaceFailures(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	require.ErrorContains(t, customprovider.CommitStagedDirectory(filepath.Join(base, "inst"), map[string][]byte{"no/such/dir.yaml": []byte("x")}), "write staged")
}

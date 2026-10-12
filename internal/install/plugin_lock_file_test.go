package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/installtest"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/governance"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadPluginLockFile_MissingFileReturnsZeroValue(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	f, err := plugincatalog.ReadPluginLockFile(dir)
	require.NoError(t, err)
	assert.Equal(t, domain.PluginLockFileSchemaVersion, f.SchemaVersion)
	assert.Empty(t, f.Inventory.Instances)
	assert.Empty(t, f.Bindings)
}

func TestReadPluginLockFile_CorruptYAMLErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, names.PluginLockFileName), []byte("not: [valid yaml"), 0o644))

	_, err := plugincatalog.ReadPluginLockFile(dir)
	require.Error(t, err)
	assert.ErrorContains(t, err, names.PluginLockFileName)
}

func TestWritePluginLockFile_RoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	want := domain.PluginLockFile{
		Lock: domain.PluginLock{
			SchemaVersion: "strategist-plugin-lock/v1",
			ResolutionID:  "resolution-1",
			GraphDigest:   "sha256:graph",
			Nodes:         []domain.PluginLockNode{{ID: "brainstorming", Kind: "adapter_contract", Digest: "digest-a"}},
		},
		Inventory: domain.PluginInventory{
			Instances: []domain.InstalledInstance{{ID: "brainstorming", State: "active", LastKnownGood: true}},
		},
		Bindings: []domain.SlotBinding{
			{SchemaVersion: "strategist-plugin-binding/v1", Slot: "discovery", InstalledInstanceID: "brainstorming", Generation: 1, Status: "enabled"},
		},
	}

	require.NoError(t, plugincatalog.WritePluginLockFile(dir, want))

	got, err := plugincatalog.ReadPluginLockFile(dir)
	require.NoError(t, err)
	assert.Equal(t, domain.PluginLockFileSchemaVersion, got.SchemaVersion)
	assert.Equal(t, want.Lock, got.Lock)
	assert.Equal(t, want.Inventory, got.Inventory)
	assert.Equal(t, want.Bindings, got.Bindings)
}

func TestWritePluginLockFile_RoundTripsBindingArtifacts(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	binding := domain.SlotBinding{
		SchemaVersion:       "strategist-plugin-binding/v1",
		Slot:                "discovery",
		InstalledInstanceID: "brainstorming@1.0.0",
		Role:                "ranger",
		WeaponVersion:       "1.0.0",
		WeaponDigest:        "sha256:weapon",
		SourceDigest:        "sha256:source",
		BindingDigest:       "sha256:binding",
		RuntimeKind:         "host",
		ConnectorID:         "host",
		Entrypoint:          "host.prompt",
		Origin:              string(domain.WeaponOriginCustom),
		Generation:          1,
		Status:              "active",
	}
	artifact, err := domain.NewWeaponBindingArtifact(binding)
	require.NoError(t, err)
	want := domain.PluginLockFile{Bindings: []domain.SlotBinding{binding}, BindingArtifacts: []domain.WeaponBindingArtifact{artifact}}

	require.NoError(t, plugincatalog.WritePluginLockFile(dir, want))
	got, err := plugincatalog.ReadPluginLockFile(dir)
	require.NoError(t, err)
	assert.Equal(t, want.Bindings, got.Bindings)
	assert.Equal(t, want.BindingArtifacts, got.BindingArtifacts)
}

func TestWritePluginLockFile_StampsSchemaVersionRegardlessOfInput(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	require.NoError(t, plugincatalog.WritePluginLockFile(dir, domain.PluginLockFile{SchemaVersion: "bogus"}))

	got, err := plugincatalog.ReadPluginLockFile(dir)
	require.NoError(t, err)
	assert.Equal(t, domain.PluginLockFileSchemaVersion, got.SchemaVersion)
}

func TestWritePluginLockFile_WriteErrorPropagates(t *testing.T) {
	t.Parallel()
	installtest.SkipIfPermissionTestUnsupported(t)
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	err := plugincatalog.WritePluginLockFile(dir, domain.PluginLockFile{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "write "+names.PluginLockFileName)
}

func TestPersistGovernanceStateRoundTripsOperatorResources(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	policy := &domain.TrustPolicy{SchemaVersion: domain.TrustPolicySchemaVersion, Revision: "r1", TrustedPublishers: []string{"acme"}, TrustedSources: []string{}}
	grants := &domain.PermissionGrantFile{SchemaVersion: domain.PermissionGrantFileSchemaVersion, Grants: []domain.PermissionGrant{{
		ID: "grant-1", PackageDigest: validDigest("1"), AdapterDigest: validDigest("2"),
		GrantedPermissions: []domain.PluginPermission{domain.PluginPermissionReadWorkspace},
	}}}
	require.NoError(t, persistGovernanceState(dir, policy, grants))

	loadedPolicy, loadedGrants, err := governance.Load(dir)
	require.NoError(t, err)
	assert.Equal(t, *policy, loadedPolicy)
	assert.Equal(t, *grants, loadedGrants)
}

func validDigest(seed string) string {
	return "sha256:" + seed + "000000000000000000000000000000000000000000000000000000000000000"
}

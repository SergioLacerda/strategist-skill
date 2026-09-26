package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/governance"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	customInstance = "fixture-provider@1.0.0"
	customDigest   = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

type customOpts struct {
	risk        string
	roles       string
	permissions string
	mode        string
	noNode      bool
}

// customWorkspace is a runtime holding a package added with `provider add`: a
// custom binding for the refinement slot, the staged package and adapter, and the
// adapter_contract lock node keyed by the package id.
func customWorkspace(t *testing.T, o customOpts) string {
	t.Helper()
	if o.risk == "" {
		o.risk = "write_analysis"
	}
	if o.roles == "" {
		o.roles = "[archivist]"
	}
	if o.permissions == "" {
		o.permissions = "[]"
	}
	if o.mode == "" {
		o.mode = "custom"
	}
	root := catalogRoot(t)
	dir := filepath.Join(root, "providers", customInstance)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	adapter := "schema_version: strategist-plugin-adapter/v1\nid: fixture-provider\nsupported_slots: [refinement]\nsupported_roles: " + o.roles + "\nentrypoints: [host.prompt]\nrequested_permissions: " + o.permissions + "\n"
	if o.risk != "none" {
		adapter += "risk_score: " + o.risk + "\n"
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "adapter.yaml"), []byte(adapter), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.yaml"), []byte("id: fixture-provider\nversion: 1.0.0\n"), 0o644))
	lock := "schema_version: strategist-plugin-lock-file/v1\nbindings:\n  - slot: refinement\n    installed_instance_id: " + customInstance + "\n    mode: " + o.mode + "\n    status: active\n"
	if !o.noNode {
		lock += "lock:\n  nodes:\n    - id: fixture-provider\n      kind: adapter_contract\n      digest: " + customDigest + "\n"
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), []byte(lock), 0o644))
	return root
}

func grantFor(t *testing.T, root string, permissions ...domain.PluginPermission) {
	t.Helper()
	require.NoError(t, governance.SaveGrants(root, domain.PermissionGrantFile{Grants: []domain.PermissionGrant{
		{SchemaVersion: domain.PermissionGrantFileSchemaVersion, ID: "grant-1", PackageDigest: customDigest, AdapterDigest: customDigest, GrantedPermissions: permissions},
	}}))
}

func TestCustomPackageResolvesByItsInstanceId(t *testing.T) {
	root := customWorkspace(t, customOpts{})

	res, errMsg := resolveSlotProvider(root, "refinement", customInstance)

	require.Empty(t, errMsg)
	assert.Equal(t, slotResolutionSkillProvider, res.kind)
	assert.Equal(t, "adapter_contract_valid", res.readiness.Descriptor.ReasonCode)
	assert.Equal(t, "custom_package_present", res.readiness.Source.ReasonCode)
	assert.Equal(t, domain.ReadinessReady, res.readiness.Entrypoint.Status)
	assert.Equal(t, "adapter_entrypoints_declared", res.readiness.Entrypoint.ReasonCode)
	assert.Equal(t, domain.ReadinessReady, res.readiness.PermissionGrant.Status)
	assert.Equal(t, "no_permissions_requested", res.readiness.PermissionGrant.ReasonCode)
	assert.False(t, res.transitionalView)
}

func TestCustomPackageRequestingPermissionsIsBlockedUntilGranted(t *testing.T) {
	root := customWorkspace(t, customOpts{permissions: "[source.write]"})

	res, errMsg := resolveSlotProvider(root, "refinement", customInstance)
	require.Empty(t, errMsg)
	assert.Equal(t, domain.ReadinessBlocked, res.readiness.PermissionGrant.Status)
	assert.Equal(t, "permission_grant_missing", res.readiness.PermissionGrant.ReasonCode)

	grantFor(t, root, "source.write")
	res, errMsg = resolveSlotProvider(root, "refinement", customInstance)
	require.Empty(t, errMsg)
	assert.Equal(t, domain.ReadinessReady, res.readiness.PermissionGrant.Status)
}

func TestCustomPackageWithoutALockDigestFailsClosed(t *testing.T) {
	root := customWorkspace(t, customOpts{noNode: true})

	res, errMsg := resolveSlotProvider(root, "refinement", customInstance)

	require.Empty(t, errMsg)
	assert.Equal(t, domain.ReadinessBlocked, res.readiness.PermissionGrant.Status, "an unknown digest is never Unknown for a custom package")
	assert.Equal(t, "custom_package_digest_missing", res.readiness.PermissionGrant.ReasonCode)
}

func TestCustomPackageBoundAsRankedStaysUnresolved(t *testing.T) {
	root := customWorkspace(t, customOpts{mode: "ranked"})

	_, errMsg := resolveSlotProvider(root, "refinement", customInstance)

	assert.Contains(t, errMsg, "not installed")
}

func TestCustomPackageWithoutRiskScoreIsBlockedBySharedPredicate(t *testing.T) {
	root := customWorkspace(t, customOpts{risk: "none"})

	_, errMsg := resolveSlotProvider(root, "refinement", customInstance)

	assert.Contains(t, errMsg, `risk_score=""`)
	assert.Contains(t, errMsg, "write_analysis")
}

func TestCustomPackageWithAMismatchedRoleIsBlocked(t *testing.T) {
	root := customWorkspace(t, customOpts{roles: "[ranger]"})
	writeDefaultRoleSlotMap(t, root)
	require.NoError(t, os.WriteFile(filepath.Join(root, "roles", "archivist.yaml"), []byte("role: archivist\nslot: refinement\nextensibility: pluggable\n"), 0o644))

	_, errMsg := resolveSlotProvider(root, "refinement", customInstance)

	assert.NotEmpty(t, errMsg)
}

func TestCustomPackageTypedByPackageIdPointsAtTheInstanceId(t *testing.T) {
	root := customWorkspace(t, customOpts{})

	_, errMsg := resolveSlotProvider(root, "refinement", "fixture-provider")

	assert.Contains(t, errMsg, "custom_package_use_instance_id")
	assert.Contains(t, errMsg, customInstance)
}

func TestLegacyBindingWithoutAStagedPackageFallsThrough(t *testing.T) {
	root := catalogRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), []byte("schema_version: strategist-plugin-lock-file/v1\nbindings:\n  - slot: refinement\n    installed_instance_id: hand-made\n    status: active\n"), 0o644))

	_, errMsg := resolveSlotProvider(root, "refinement", "hand-made")

	assert.Contains(t, errMsg, "not installed", "no providers/ package: the other resolvers answer, as before")
}

package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const weaponCatalog = `schema_version: strategist-plugin-catalog/v2
providers:
  - id: host-weapon
    risk_score: write_analysis
    compatibility_source: embedded
    canonical_role: archivist
    roles: [archivist]
    runtime:
      kind: host
      host_api: strategist-host-skill/v1
  - id: rooted-weapon
    risk_score: write_analysis
    compatibility_source: embedded
    canonical_role: archivist
    roles: [archivist]
    runtime:
      kind: openspec_root
      root: .strategist/openspec
  - id: sniper
    risk_score: controlled
    compatibility_source: native_role
  - id: wrong-risk
    risk_score: controlled
    compatibility_source: embedded
    canonical_role: archivist
    roles: [archivist]
    runtime:
      kind: host
`

// catalogRoot builds a runtime with a catalog and NO compat view.
func catalogRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".strategist")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "plugins"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins", "catalog.yaml"), []byte(weaponCatalog), 0o644))
	return root
}

func writePayload(t *testing.T, root, provider string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "skills", provider), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "skills", provider, "SKILL.md"), []byte("# skill\n"), 0o644))
}

func TestCatalogWeaponResolvesWithoutAnyCompatView(t *testing.T) {
	root := catalogRoot(t)
	writePayload(t, root, "host-weapon")

	res, errMsg := resolveSlotProvider(root, "refinement", "host-weapon")

	require.Empty(t, errMsg)
	assert.Equal(t, slotResolutionSkillProvider, res.kind)
	assert.Equal(t, "catalog_entry_valid", res.readiness.Descriptor.ReasonCode)
	assert.Equal(t, "catalog_entry_present", res.readiness.Source.ReasonCode)
	assert.Equal(t, domain.ReadinessReady, res.readiness.Entrypoint.Status)
	assert.Equal(t, "entrypoint_payload_present", res.readiness.Entrypoint.ReasonCode)
	_, statErr := os.Stat(filepath.Join(root, "skills", "host-weapon", "skill.yaml"))
	require.ErrorIs(t, statErr, os.ErrNotExist, "the fixture really has no compat view")
}

func TestCatalogWeaponEntrypointBlocksWhenThePayloadIsMissingOrEmpty(t *testing.T) {
	root := catalogRoot(t)

	res, errMsg := resolveSlotProvider(root, "refinement", "host-weapon")
	require.Empty(t, errMsg, "resolution succeeds; the readiness dimension carries the block")
	assert.Equal(t, domain.ReadinessBlocked, res.readiness.Entrypoint.Status)
	assert.Equal(t, "entrypoint_payload_missing", res.readiness.Entrypoint.ReasonCode)

	require.NoError(t, os.MkdirAll(filepath.Join(root, "skills", "host-weapon"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "skills", "host-weapon", "SKILL.md"), nil, 0o644))
	res, _ = resolveSlotProvider(root, "refinement", "host-weapon")
	assert.Equal(t, "entrypoint_payload_empty", res.readiness.Entrypoint.ReasonCode)
}

func TestCatalogWeaponEntrypointForAnOpenspecRootChecksTheRuntimeRoot(t *testing.T) {
	root := catalogRoot(t)

	res, errMsg := resolveSlotProvider(root, "refinement", "rooted-weapon")
	require.Empty(t, errMsg)
	assert.Equal(t, domain.ReadinessBlocked, res.readiness.Entrypoint.Status)
	assert.Equal(t, "runtime_root_missing", res.readiness.Entrypoint.ReasonCode)

	require.NoError(t, os.MkdirAll(filepath.Join(root, "openspec"), 0o755))
	res, _ = resolveSlotProvider(root, "refinement", "rooted-weapon")
	assert.Equal(t, domain.ReadinessReady, res.readiness.Entrypoint.Status)
	assert.Equal(t, "runtime_root_present", res.readiness.Entrypoint.ReasonCode)
}

func TestCatalogWeaponRiskMustMatchTheSlotContract(t *testing.T) {
	root := catalogRoot(t)
	writePayload(t, root, "wrong-risk")

	_, errMsg := resolveSlotProvider(root, "refinement", "wrong-risk")

	assert.Contains(t, errMsg, `risk_score="controlled" but slot requires "write_analysis"`)
}

func TestCatalogNativeRoleEntryTakesTheNativeBranchWithoutAFileProbe(t *testing.T) {
	root := catalogRoot(t)
	require.NoError(t, os.MkdirAll(filepath.Join(root, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "roles", "sniper.yaml"), []byte("role: sniper\nslot: execution\nextensibility: pluggable\n"), 0o644))

	res, errMsg := resolveSlotProvider(root, "execution", "sniper")

	require.Empty(t, errMsg)
	assert.Equal(t, slotResolutionNativeRole, res.kind)
}

// U-01 (see .analysis/pending/20260925-u01-provider-add-check-resolution.md): a package
// added with `provider add` is not resolved by `check`. DEC-010 step 2 is documented
// but not implemented; this pins the known gap so closing it is a deliberate change.
func TestProviderAddPackageIsStillNotResolvedByCheck(t *testing.T) {
	root := catalogRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), []byte("schema_version: strategist-plugin-lock-file/v1\nbindings:\n  - slot: refinement\n    installed_instance_id: fixture-provider@1.0.0\n    mode: custom\n    status: active\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "providers", "fixture-provider@1.0.0"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "providers", "fixture-provider@1.0.0", "adapter.yaml"), []byte("risk_score: write_analysis\nsupported_roles: [archivist]\n"), 0o644))

	_, errMsg := resolveSlotProvider(root, "refinement", "fixture-provider")

	assert.Contains(t, errMsg, "not installed")
}

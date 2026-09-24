package check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillProviderTrustReadiness_NoPolicyConfiguredIsReady(t *testing.T) {
	root := t.TempDir()

	got := skillProviderTrustReadiness(root, "brainstorming", "sha256:abc")
	assert.Equal(t, domain.ReadinessReady, got.Status)
	assert.Equal(t, "no_trust_policy_configured", got.ReasonCode)
}

func TestSkillProviderTrustReadiness_RealPolicyBlocksUntrustedPublisher(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(root, "trust-policy.yaml"),
		[]byte("schema_version: strategist-trust-policy/v1\nrevision: r1\ntrusted_publishers: [acme]\n"),
		0o644,
	))

	got := skillProviderTrustReadiness(root, "brainstorming", "sha256:abc")
	assert.Equal(t, domain.ReadinessBlocked, got.Status)
	assert.Equal(t, "publisher_not_trusted", got.ReasonCode)
}

func TestSkillProviderTrustReadiness_MissingFileFallsBackToEmptyPolicy(t *testing.T) {
	root := t.TempDir()

	policy := readTrustPolicy(root)
	assert.Equal(t, domain.TrustPolicy{}, policy)
}

func TestReadTrustPolicy_InvalidFileFallsBackToEmptyPolicy(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "trust-policy.yaml"), []byte(": invalid: yaml\n"), 0o644))

	assert.Equal(t, domain.TrustPolicy{}, readTrustPolicy(root))
}

func TestReadPluginsLockFile_InvalidFileFallsBackToEmptyLock(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), []byte(": invalid: yaml\n"), 0o644))

	assert.Empty(t, readPluginsLockFile(root).Bindings)
}

func TestSkillProviderPermissionGrantReadiness_NoDigestIsUnknown(t *testing.T) {
	got := skillProviderPermissionGrantReadiness("")
	assert.Equal(t, domain.ReadinessUnknown, got.Status)
	assert.Equal(t, "permission_grant_not_evaluated", got.ReasonCode)
}

func TestSkillProviderPermissionGrantReadiness_NoPermissionsRequestedIsReady(t *testing.T) {
	got := skillProviderPermissionGrantReadiness("sha256:1111111111111111111111111111111111111111111111111111111111111111")
	assert.Equal(t, domain.ReadinessReady, got.Status)
	assert.Equal(t, "no_permissions_requested", got.ReasonCode)
}

func TestSkillProviderPermissionGrantReadiness_UsesPersistedGrant(t *testing.T) {
	root := t.TempDir()
	digest := "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	require.NoError(t, os.WriteFile(filepath.Join(root, "permission-grants.yaml"), []byte(`schema_version: strategist-permission-grants/v1
grants:
  - id: grant-1
    package_digest: `+digest+`
    adapter_digest: `+digest+`
    granted_permissions: [workspace.read]
`), 0o644))

	got := skillProviderPermissionGrantReadinessFor(root, digest, []domain.PluginPermission{domain.PluginPermissionReadWorkspace})
	assert.Equal(t, domain.ReadinessReady, got.Status)
}

func TestSkillProviderPermissionGrantReadiness_MissingPersistedGrantBlocksRequestedPermission(t *testing.T) {
	root := t.TempDir()
	digest := "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	got := skillProviderPermissionGrantReadinessFor(root, digest, []domain.PluginPermission{domain.PluginPermissionReadWorkspace})
	assert.Equal(t, domain.ReadinessBlocked, got.Status)
	assert.Equal(t, "permission_grant_missing", got.ReasonCode)
}

// TestCheckCmd_JSON_TrustAndGrantWiredFromRealLockDigest is the P4/P5/P6
// integration regression test: with a real adapter_contract digest present
// in plugins.lock, both Trust and PermissionGrant must move from the old
// hardcoded "*_not_evaluated" Unknown status to a genuinely computed Ready
// result, without changing the overall check outcome.
func TestCheckCmd_JSON_TrustAndGrantWiredFromRealLockDigest(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugins.lock"), []byte(`schema_version: strategist-plugin-lock-file/v1
lock:
  nodes:
    - id: brainstorming
      kind: adapter_contract
      digest: sha256:1111111111111111111111111111111111111111111111111111111111111111
bindings:
  - slot: discovery
    installed_instance_id: brainstorming
    status: enabled
  - slot: refinement
    installed_instance_id: openspec-explore
    status: enabled
`), 0o644))
	checkRoot = dir
	checkJSON = true

	out := captureStdout(t, func() {
		err := checkCmd.RunE(checkCmd, nil)
		require.NoError(t, err)
	})

	var result domain.PreflightResult
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "ready", result.Status)
}

// TestCheckCmd_JSON_RankedBindingReadyByCertification is the task-3.3
// regression test: a mode: ranked binding must report Trust/PermissionGrant
// as Ready-by-certification (sourced from the catalog's certification
// stamp), never falling through to trust.Verify/policy.EvaluateGrant's
// empty-policy Ready reason.
func TestCheckCmd_JSON_RankedBindingReadyByCertification(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugins.lock"), []byte(`schema_version: strategist-plugin-lock-file/v1
bindings:
  - slot: discovery
    installed_instance_id: brainstorming
    mode: ranked
    status: enabled
  - slot: refinement
    installed_instance_id: openspec-explore
    status: enabled
`), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "plugins"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugins", "catalog.yaml"), []byte(`schema_version: strategist-plugin-catalog/v2
providers:
  - id: brainstorming
    canonical_role: ranger
    roles: [ranger]
    ranked: true
    certification_digest: sha256:1111111111111111111111111111111111111111111111111111111111111111
    host_api_digest: sha256:2222222222222222222222222222222222222222222222222222222222222222
    connector_digest: sha256:3333333333333333333333333333333333333333333333333333333333333333
    test_suite_digest: sha256:4444444444444444444444444444444444444444444444444444444444444444
    conformance_level: C1
`), 0o644))
	checkRoot = dir
	checkJSON = true

	out := captureStdout(t, func() {
		err := checkCmd.RunE(checkCmd, nil)
		require.NoError(t, err)
	})

	var result domain.PreflightResult
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "ready", result.Status)

	trustCheck, grantCheck, _ := rankedCertificationReadiness(dir, "discovery", "brainstorming")
	assert.Equal(t, domain.ReadinessReady, trustCheck.Status)
	assert.Equal(t, "ready_by_certification", trustCheck.ReasonCode)
	assert.Equal(t, domain.ReadinessReady, grantCheck.Status)
	assert.Equal(t, "ready_by_certification", grantCheck.ReasonCode)
}

// TestCheckCmd_JSON_RankedBindingReadyByCertification_RefinementSlot is
// the 20260916-ranked-skills-end-to-end-evaluation Layer 4 coverage-gap
// fix: the refinement-slot counterpart to
// TestCheckCmd_JSON_RankedBindingReadyByCertification, which only ever
// exercised mode: ranked on the discovery binding.
func TestCheckCmd_JSON_RankedBindingReadyByCertification_RefinementSlot(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "active.yaml"),
		[]byte("mode: epic\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n  refinement: openspec-propose\n  execution: sdd-ask\n"),
		0o644,
	))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugins.lock"), []byte(`schema_version: strategist-plugin-lock-file/v1
bindings:
  - slot: discovery
    installed_instance_id: brainstorming
    status: enabled
  - slot: refinement
    installed_instance_id: openspec-propose
    mode: ranked
    status: enabled
`), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "plugins"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugins", "catalog.yaml"), []byte(`schema_version: strategist-plugin-catalog/v2
providers:
  - id: openspec-propose
    canonical_role: archivist
    roles: [archivist]
    ranked: true
    certification_digest: sha256:1111111111111111111111111111111111111111111111111111111111111111
    host_api_digest: sha256:2222222222222222222222222222222222222222222222222222222222222222
    connector_digest: sha256:3333333333333333333333333333333333333333333333333333333333333333
    test_suite_digest: sha256:4444444444444444444444444444444444444444444444444444444444444444
    conformance_level: C1
`), 0o644))
	checkRoot = dir
	checkJSON = true

	out := captureStdout(t, func() {
		err := checkCmd.RunE(checkCmd, nil)
		require.NoError(t, err)
	})

	var result domain.PreflightResult
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "ready", result.Status)

	trustCheck, grantCheck, _ := rankedCertificationReadiness(dir, "refinement", "openspec-propose")
	assert.Equal(t, domain.ReadinessReady, trustCheck.Status)
	assert.Equal(t, "ready_by_certification", trustCheck.ReasonCode)
	assert.Equal(t, domain.ReadinessReady, grantCheck.Status)
	assert.Equal(t, "ready_by_certification", grantCheck.ReasonCode)
}

func TestBindingIsRanked(t *testing.T) {
	lock := domain.PluginLockFile{Bindings: []domain.SlotBinding{
		{Slot: "discovery", InstalledInstanceID: "brainstorming", Mode: domain.SlotBindingModeRanked},
		{Slot: "refinement", InstalledInstanceID: "openspec-explore"},
	}}
	assert.True(t, bindingIsRanked(lock, "discovery", "brainstorming"))
	assert.False(t, bindingIsRanked(lock, "refinement", "openspec-explore"))
	assert.False(t, bindingIsRanked(lock, "discovery", "other-provider"))
}

func TestRankedCertificationReadiness_BlocksWhenNotCertified(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "plugins"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugins", "catalog.yaml"), []byte(`schema_version: strategist-plugin-catalog/v2
providers:
  - id: brainstorming
    canonical_role: ranger
`), 0o644))

	trustCheck, grantCheck, _ := rankedCertificationReadiness(dir, "discovery", "brainstorming")
	assert.Equal(t, domain.ReadinessBlocked, trustCheck.Status)
	assert.Equal(t, "ranked_not_certified", trustCheck.ReasonCode)
	assert.Equal(t, domain.ReadinessBlocked, grantCheck.Status)
}

func TestRankedCertificationReadiness_BlocksWhenCatalogMissing(t *testing.T) {
	dir := t.TempDir()

	trustCheck, grantCheck, _ := rankedCertificationReadiness(dir, "discovery", "brainstorming")
	assert.Equal(t, domain.ReadinessBlocked, trustCheck.Status)
	assert.Equal(t, "ranked_catalog_unreadable", trustCheck.ReasonCode)
	assert.Equal(t, domain.ReadinessBlocked, grantCheck.Status)
}

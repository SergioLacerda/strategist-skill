package check

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveSlotProvider_SkillYAMLUnreadablePermission(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod not reliable on windows")
	}
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "skills", "brainstorming")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	skillPath := filepath.Join(skillDir, "skill.yaml")
	require.NoError(t, os.WriteFile(skillPath, []byte("id: brainstorming\n"), 0o000))
	t.Cleanup(func() { _ = os.Chmod(skillPath, 0o644) })
	if os.Getuid() == 0 {
		t.Skip("running as root — file permission checks do not apply")
	}

	_, errMsg := resolveSlotProvider(dir, "discovery", "brainstorming")
	assert.Contains(t, errMsg, "read")
	assert.Contains(t, errMsg, skillPath)
}

func TestResolveSkillProviderSlot_InvalidYAML(t *testing.T) {
	_, errMsg := resolveSkillProviderSlot(t.TempDir(), "discovery", "brainstorming", "/tmp/skill.yaml", []byte("id: [unterminated\n"))
	assert.Contains(t, errMsg, "skill.yaml invalid")
}

func TestResolveNativeRoleSlot_UnreadablePermission(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod not reliable on windows")
	}
	dir := t.TempDir()
	rolesDir := filepath.Join(dir, "roles")
	require.NoError(t, os.MkdirAll(rolesDir, 0o755))
	rolePath := filepath.Join(rolesDir, "sniper.yaml")
	require.NoError(t, os.WriteFile(rolePath, []byte("role: sniper\nslot: execution\n"), 0o000))
	t.Cleanup(func() { _ = os.Chmod(rolePath, 0o644) })
	if os.Getuid() == 0 {
		t.Skip("running as root — file permission checks do not apply")
	}

	_, errMsg := resolveNativeRoleSlot(dir, "execution", "sniper", filepath.Join(dir, "skills", "sniper", "skill.yaml"))
	assert.Contains(t, errMsg, "unreadable")
}

func TestResolveNativeRoleSlot_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	rolesDir := filepath.Join(dir, "roles")
	require.NoError(t, os.MkdirAll(rolesDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rolesDir, "sniper.yaml"), []byte("role: [unterminated\n"), 0o644))

	_, errMsg := resolveNativeRoleSlot(dir, "execution", "sniper", filepath.Join(dir, "skills", "sniper", "skill.yaml"))
	assert.Contains(t, errMsg, "malformed YAML")
}

func TestResolveSkillProviderSlot_AttachesUnsupportedReadiness(t *testing.T) {
	t.Parallel()

	// skillPath is fabricated and never written to disk, so the entrypoint
	// probe (probeSkillEntrypoint) correctly reports the file as missing
	// rather than the old hardcoded "entrypoint_probe_unsupported" — the
	// probe is now a real static check, not a no-op.
	res, errMsg := resolveSkillProviderSlot(t.TempDir(), "discovery", "brainstorming", "skills/brainstorming/skill.yaml", []byte("risk_score: write_analysis\n"))
	require.Empty(t, errMsg)

	assert.Equal(t, slotResolutionSkillProvider, res.kind)
	assert.False(t, res.readiness.Ready())
	assert.Contains(t, res.readiness.ReasonCodes(), "connector_unsupported")
	assert.Contains(t, res.readiness.ReasonCodes(), "entrypoint_file_missing")
}

func TestResolveSkillProviderSlot_EntrypointProbeVerifiesRealManifest(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	skillDir := filepath.Join(dir, "skills", "brainstorming")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	skillPath := filepath.Join(skillDir, "skill.yaml")
	require.NoError(t, os.WriteFile(skillPath, []byte("id: brainstorming\nrisk_score: write_analysis\n"), 0o644))

	res, errMsg := resolveSkillProviderSlot(dir, "discovery", "brainstorming", skillPath, []byte("risk_score: write_analysis\n"))
	require.Empty(t, errMsg)

	assert.Equal(t, domain.ReadinessReady, res.readiness.Entrypoint.Status)
	assert.Equal(t, "entrypoint_manifest_verified", res.readiness.Entrypoint.ReasonCode)
}

func TestResolveSkillProviderSlot_EntrypointProbeBlocksIDMismatch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	skillDir := filepath.Join(dir, "skills", "brainstorming")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	skillPath := filepath.Join(skillDir, "skill.yaml")
	require.NoError(t, os.WriteFile(skillPath, []byte("id: some-other-id\nrisk_score: write_analysis\n"), 0o644))

	res, errMsg := resolveSkillProviderSlot(dir, "discovery", "brainstorming", skillPath, []byte("risk_score: write_analysis\n"))
	require.Empty(t, errMsg)

	assert.Equal(t, domain.ReadinessBlocked, res.readiness.Entrypoint.Status)
	assert.Equal(t, "entrypoint_id_mismatch", res.readiness.Entrypoint.ReasonCode)
}

func TestProbeSkillEntrypoint_EmptyFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "skill.yaml")
	require.NoError(t, os.WriteFile(path, []byte{}, 0o644))

	check := probeSkillEntrypoint("brainstorming", path)
	assert.Equal(t, domain.ReadinessBlocked, check.Status)
	assert.Equal(t, "entrypoint_file_empty", check.ReasonCode)
}

func TestProbeSkillEntrypoint_UnparseableYAML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "skill.yaml")
	require.NoError(t, os.WriteFile(path, []byte("id: [unterminated\n"), 0o644))

	check := probeSkillEntrypoint("brainstorming", path)
	assert.Equal(t, domain.ReadinessBlocked, check.Status)
	assert.Equal(t, "entrypoint_manifest_unparseable", check.ReasonCode)
}

func TestProbeSkillEntrypoint_MissingID(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "skill.yaml")
	require.NoError(t, os.WriteFile(path, []byte("risk_score: write_analysis\n"), 0o644))

	check := probeSkillEntrypoint("brainstorming", path)
	assert.Equal(t, domain.ReadinessBlocked, check.Status)
	assert.Equal(t, "entrypoint_id_missing", check.ReasonCode)
}

// --- blockedReadinessErrors ---

func TestBlockedReadinessErrors_ReportsOnlyBlockedDimensions(t *testing.T) {
	t.Parallel()

	vector := domain.PluginReadinessVector{
		Descriptor:          domain.ReadinessCheck{Status: domain.ReadinessReady},
		Source:              domain.ReadinessCheck{Status: domain.ReadinessReady},
		Trust:               domain.ReadinessCheck{Status: domain.ReadinessUnknown, ReasonCode: "trust_policy_not_evaluated"},
		Dependencies:        domain.ReadinessCheck{Status: domain.ReadinessUnknown},
		HostAPI:             domain.ReadinessCheck{Status: domain.ReadinessUnknown},
		Connector:           domain.ReadinessCheck{Status: domain.ReadinessUnsupported, ReasonCode: "connector_unsupported"},
		Entrypoint:          domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "entrypoint_id_mismatch", Detail: "manifest id mismatch"},
		PermissionGrant:     domain.ReadinessCheck{Status: domain.ReadinessUnknown},
		EnforcementCoverage: domain.ReadinessCheck{Status: domain.ReadinessUnsupported},
		ActiveBinding:       domain.ReadinessCheck{Status: domain.ReadinessReady},
	}

	errs := blockedReadinessErrors("execution", vector)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "slot execution")
	assert.Contains(t, errs[0], "entrypoint")
	assert.Contains(t, errs[0], "entrypoint_id_mismatch")
	assert.Contains(t, errs[0], "manifest id mismatch")
}

func TestBlockedReadinessErrors_NoBlockedDimensionsReturnsEmpty(t *testing.T) {
	t.Parallel()

	vector := domain.PluginReadinessVector{
		Trust:      domain.ReadinessCheck{Status: domain.ReadinessUnknown},
		Connector:  domain.ReadinessCheck{Status: domain.ReadinessUnsupported},
		Entrypoint: domain.ReadinessCheck{Status: domain.ReadinessReady},
	}

	assert.Empty(t, blockedReadinessErrors("discovery", vector))
}

func TestResolveNativeRoleSlot_AttachesNativeReadiness(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rolesDir := filepath.Join(dir, "roles")
	require.NoError(t, os.MkdirAll(rolesDir, 0o755))
	rolePath := filepath.Join(rolesDir, "sniper.yaml")
	require.NoError(t, os.WriteFile(rolePath, []byte("role: sniper\nslot: execution\n"), 0o644))

	res, errMsg := resolveNativeRoleSlot(dir, "execution", "sniper", filepath.Join(dir, "skills", "sniper", "skill.yaml"))
	require.Empty(t, errMsg)

	assert.Equal(t, slotResolutionNativeRole, res.kind)
	assert.False(t, res.readiness.Ready(), "native readiness must still report unsupported enforcement explicitly")
	assert.Contains(t, res.readiness.ReasonCodes(), "enforcement_unsupported")
	assert.Equal(t, rolePath, res.readiness.Descriptor.Detail)
}

func TestSlotResolutionKindUsesPluginVocabulary(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "external skill plugin", slotResolutionSkillProvider.label())
	assert.Equal(t, "native role", slotResolutionNativeRole.label())
}

func TestCheckRoleProviderCompatibility_NoRoleSlotMap(t *testing.T) {
	t.Parallel()
	dir := t.TempDir() // no roles/default.yaml at all
	errMsg := checkRoleProviderCompatibility(dir, "refinement", "openspec-explore", "write_analysis",
		[]byte("id: openspec-explore\ncanonical_role: archivist\n"))
	assert.Empty(t, errMsg)
}

func TestCheckRoleProviderCompatibility_SlotNotMappedToARole(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	rolesDir := filepath.Join(dir, "roles")
	require.NoError(t, os.MkdirAll(rolesDir, 0o755))
	// default.yaml exists but has no entry for "execution".
	require.NoError(t, os.WriteFile(filepath.Join(rolesDir, "default.yaml"),
		[]byte("discovery: ranger\nrefinement: archivist\n"), 0o644))

	errMsg := checkRoleProviderCompatibility(dir, "execution", "sdd-ask", "controlled",
		[]byte("id: sdd-ask\n"))
	assert.Empty(t, errMsg)
}

func TestCheckRoleProviderCompatibility_RoleFileMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	rolesDir := filepath.Join(dir, "roles")
	require.NoError(t, os.MkdirAll(rolesDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rolesDir, "default.yaml"),
		[]byte("discovery: ranger\nrefinement: archivist\nexecution: sniper\n"), 0o644))
	// roles/archivist.yaml deliberately absent.

	errMsg := checkRoleProviderCompatibility(dir, "refinement", "openspec-explore", "write_analysis",
		[]byte("id: openspec-explore\ncanonical_role: archivist\n"))
	assert.Empty(t, errMsg)
}

func TestCheckRoleProviderCompatibility_SkillDeclaresNoCanonicalRole(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	rolesDir := filepath.Join(dir, "roles")
	require.NoError(t, os.MkdirAll(rolesDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rolesDir, "default.yaml"),
		[]byte("discovery: ranger\nrefinement: archivist\nexecution: sniper\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(rolesDir, "archivist.yaml"),
		[]byte("role: archivist\nslot: refinement\n"), 0o644))

	// sdd-ask declares no canonical_role — not every provider is expected to.
	errMsg := checkRoleProviderCompatibility(dir, "refinement", "sdd-ask", "controlled",
		[]byte("id: sdd-ask\n"))
	assert.Empty(t, errMsg)
}

func TestCheckRoleProviderCompatibility_MatchingCanonicalRole(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	rolesDir := filepath.Join(dir, "roles")
	require.NoError(t, os.MkdirAll(rolesDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rolesDir, "default.yaml"),
		[]byte("discovery: ranger\nrefinement: archivist\nexecution: sniper\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(rolesDir, "archivist.yaml"),
		[]byte("role: archivist\nslot: refinement\n"), 0o644))

	errMsg := checkRoleProviderCompatibility(dir, "refinement", "openspec-propose", "write_analysis",
		[]byte("id: openspec-propose\ncanonical_role: archivist\n"))
	assert.Empty(t, errMsg)
}

func TestCheckRoleProviderCompatibility_MismatchedCanonicalRole(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	rolesDir := filepath.Join(dir, "roles")
	require.NoError(t, os.MkdirAll(rolesDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rolesDir, "default.yaml"),
		[]byte("discovery: ranger\nrefinement: archivist\nexecution: sniper\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(rolesDir, "archivist.yaml"),
		[]byte("role: archivist\nslot: refinement\n"), 0o644))

	// A provider configured for the refinement slot but whose own
	// canonical_role claims "ranger" (discovery's role) instead of
	// "archivist" — this is the exact gap this check closes.
	errMsg := checkRoleProviderCompatibility(dir, "refinement", "mis-wired-provider", "write_analysis",
		[]byte("id: mis-wired-provider\ncanonical_role: ranger\n"))
	require.NotEmpty(t, errMsg)
	assert.Contains(t, errMsg, "role-incompatible")
	assert.Contains(t, errMsg, "role_mismatch")
	assert.Contains(t, errMsg, "mis-wired-provider")
}

// TestResolveSlotProvider_RoleIncompatibleProviderBlocksSlotResolution is the
// end-to-end version of the mismatch case above, through the same
// resolveSlotProvider entrypoint checkCmd itself calls, proving the
// incompatibility actually blocks slot resolution rather than only being
// detectable via the unexported helper directly.
func TestResolveSlotProvider_RoleIncompatibleProviderBlocksSlotResolution(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	rolesDir := filepath.Join(dir, "roles")
	require.NoError(t, os.MkdirAll(rolesDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rolesDir, "default.yaml"),
		[]byte("discovery: ranger\nrefinement: archivist\nexecution: sniper\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(rolesDir, "archivist.yaml"),
		[]byte("role: archivist\nslot: refinement\n"), 0o644))
	skillDir := filepath.Join(dir, "skills", "mis-wired-provider")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "skill.yaml"),
		[]byte("id: mis-wired-provider\nrisk_score: write_analysis\ncanonical_role: ranger\n"), 0o644))

	_, errMsg := resolveSlotProvider(dir, "refinement", "mis-wired-provider")
	require.NotEmpty(t, errMsg)
	assert.Contains(t, errMsg, "role-incompatible")
}

func TestResolveSlotProvider_CatalogedProviderWithoutHandoffSchemaUsesRoleCheckpoint(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	rolesDir := filepath.Join(dir, "roles")
	require.NoError(t, os.MkdirAll(rolesDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rolesDir, "default.yaml"),
		[]byte("discovery: ranger\nrefinement: archivist\nexecution: sniper\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(rolesDir, "archivist.yaml"),
		[]byte("role: archivist\nslot: refinement\n"), 0o644))
	skillDir := filepath.Join(dir, "skills", "openspec-propose")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "skill.yaml"),
		[]byte("id: openspec-propose\nrisk_score: write_analysis\ncanonical_role: archivist\n"), 0o644))
	pluginsDir := filepath.Join(dir, "plugins")
	require.NoError(t, os.MkdirAll(pluginsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pluginsDir, "catalog.yaml"), []byte(`schema_version: strategist-plugin-catalog/v1
providers:
  - id: openspec-propose
    canonical_role: archivist
    compatibility_source: embedded
    risk_score: write_analysis
`), 0o644))

	_, errMsg := resolveSlotProvider(dir, "refinement", "openspec-propose")
	assert.Empty(t, errMsg)
}

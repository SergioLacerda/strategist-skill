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
	_, errMsg := resolveSkillProviderSlot("discovery", "brainstorming", "/tmp/skill.yaml", []byte("id: [unterminated\n"))
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
	res, errMsg := resolveSkillProviderSlot("discovery", "brainstorming", "skills/brainstorming/skill.yaml", []byte("risk_score: write_analysis\n"))
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

	res, errMsg := resolveSkillProviderSlot("discovery", "brainstorming", skillPath, []byte("risk_score: write_analysis\n"))
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

	res, errMsg := resolveSkillProviderSlot("discovery", "brainstorming", skillPath, []byte("risk_score: write_analysis\n"))
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

// --- resolveNativeFallback (ADR-0028) ---

func TestResolveNativeFallback_CompatibleRoleFound(t *testing.T) {
	dir := t.TempDir()
	rolesDir := filepath.Join(dir, "roles")
	require.NoError(t, os.MkdirAll(rolesDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rolesDir, "default.yaml"),
		[]byte("discovery: ranger\nrefinement: archivist\nexecution: sniper\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(rolesDir, "archivist.yaml"),
		[]byte("role: archivist\nslot: refinement\n"), 0o644))

	provider, path := resolveNativeFallback(dir, "refinement")
	assert.Equal(t, "archivist", provider)
	assert.Equal(t, filepath.Join(rolesDir, "archivist.yaml"), path)
}

func TestResolveNativeFallback_NoDefaultMap(t *testing.T) {
	dir := t.TempDir()
	provider, path := resolveNativeFallback(dir, "refinement")
	assert.Empty(t, provider)
	assert.Empty(t, path)
}

func TestResolveNativeFallback_DefaultMapMalformedYAML(t *testing.T) {
	dir := t.TempDir()
	rolesDir := filepath.Join(dir, "roles")
	require.NoError(t, os.MkdirAll(rolesDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rolesDir, "default.yaml"), []byte("discovery: [unterminated\n"), 0o644))

	provider, path := resolveNativeFallback(dir, "refinement")
	assert.Empty(t, provider)
	assert.Empty(t, path)
}

func TestResolveNativeFallback_SlotMissingFromDefaultMap(t *testing.T) {
	dir := t.TempDir()
	rolesDir := filepath.Join(dir, "roles")
	require.NoError(t, os.MkdirAll(rolesDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rolesDir, "default.yaml"), []byte("discovery: ranger\n"), 0o644))

	provider, path := resolveNativeFallback(dir, "refinement")
	assert.Empty(t, provider)
	assert.Empty(t, path)
}

func TestResolveNativeFallback_CandidateRoleFileMissing(t *testing.T) {
	dir := t.TempDir()
	rolesDir := filepath.Join(dir, "roles")
	require.NoError(t, os.MkdirAll(rolesDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rolesDir, "default.yaml"),
		[]byte("discovery: ranger\nrefinement: archivist\nexecution: sniper\n"), 0o644))
	// archivist.yaml deliberately absent — no fallback should be reported.

	provider, path := resolveNativeFallback(dir, "refinement")
	assert.Empty(t, provider)
	assert.Empty(t, path)
}

func TestResolveNativeFallback_CandidateRoleSlotMismatch(t *testing.T) {
	dir := t.TempDir()
	rolesDir := filepath.Join(dir, "roles")
	require.NoError(t, os.MkdirAll(rolesDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rolesDir, "default.yaml"),
		[]byte("discovery: ranger\nrefinement: archivist\nexecution: sniper\n"), 0o644))
	// archivist.yaml declares the wrong slot — resolveNativeRoleSlot's own
	// validation must reject it, so no fallback is reported (never a role file
	// that is present but structurally incompatible).
	require.NoError(t, os.WriteFile(filepath.Join(rolesDir, "archivist.yaml"),
		[]byte("role: archivist\nslot: discovery\n"), 0o644))

	provider, path := resolveNativeFallback(dir, "refinement")
	assert.Empty(t, provider)
	assert.Empty(t, path)
}

package check

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

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

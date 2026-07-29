package main

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

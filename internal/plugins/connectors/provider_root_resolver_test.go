package connectors

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const resolverSkill = "---\nname: sample-skill\nmetadata:\n  version: \"1.0.0\"\n---\nbody\n"

func TestResolveCustomProviderPackagePrefersWorkspaceAgentsOverGlobal(t *testing.T) {
	workspace := t.TempDir()
	home := t.TempDir()
	writeResolverSkill(t, filepath.Join(workspace, ".agents", "skills", "sample-skill"), resolverSkill)
	writeResolverSkill(t, filepath.Join(home, ".agents", "skills", "sample-skill"), resolverSkill+"global\n")

	resolved, err := ResolveCustomProviderPackage(workspace, "sample-skill", DefaultGlobalProviderRoots(home))
	require.NoError(t, err)
	require.Equal(t, ProviderOriginWorkspaceAgents, resolved.Origin)
	require.Equal(t, filepath.Join(workspace, ".agents", "skills", "sample-skill"), resolved.Path)
	require.Equal(t, filepath.Join(resolved.Path, "SKILL.md"), resolved.Entrypoint)
}

func TestResolveCustomProviderPackageUsesWorkspaceCodexBeforeGlobal(t *testing.T) {
	workspace := t.TempDir()
	home := t.TempDir()
	writeResolverSkill(t, filepath.Join(workspace, ".codex", "skills", "sample-skill"), resolverSkill)
	writeResolverSkill(t, filepath.Join(home, ".agents", "skills", "sample-skill"), resolverSkill+"global\n")

	resolved, err := ResolveCustomProviderPackage(workspace, "sample-skill", DefaultGlobalProviderRoots(home))
	require.NoError(t, err)
	require.Equal(t, ProviderOriginWorkspaceCodex, resolved.Origin)
}

func TestResolveCustomProviderPackageRejectsInvalidLocalWithoutGlobalFallback(t *testing.T) {
	workspace := t.TempDir()
	home := t.TempDir()
	local := filepath.Join(workspace, ".agents", "skills", "sample-skill")
	require.NoError(t, os.MkdirAll(local, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(local, "SKILL.md"), []byte("not frontmatter"), 0o644))
	writeResolverSkill(t, filepath.Join(home, ".agents", "skills", "sample-skill"), resolverSkill)

	_, err := ResolveCustomProviderPackage(workspace, "sample-skill", DefaultGlobalProviderRoots(home))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid")
}

func TestResolveCustomProviderPackageUsesGlobalWhenLocalAbsent(t *testing.T) {
	workspace := t.TempDir()
	home := t.TempDir()
	writeResolverSkill(t, filepath.Join(home, ".codex", "skills", "sample-skill"), resolverSkill)

	resolved, err := ResolveCustomProviderPackage(workspace, "sample-skill", DefaultGlobalProviderRoots(home))
	require.NoError(t, err)
	require.Equal(t, ProviderOriginGlobalCodex, resolved.Origin)
	require.NotEmpty(t, resolved.Package.Digest)
	require.Equal(t, resolved.Path, resolved.SeedPath)
}

func writeResolverSkill(t *testing.T, dir, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644))
}

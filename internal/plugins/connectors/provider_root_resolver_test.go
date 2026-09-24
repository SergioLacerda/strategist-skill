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

func TestResolveCustomProviderPackageRejectsIncompleteInputsAndMisses(t *testing.T) {
	workspace := t.TempDir()
	_, err := ResolveCustomProviderPackage(workspace, " ", nil)
	require.ErrorContains(t, err, "provider id is empty")
	_, err = ResolveCustomProviderPackage(" ", "sample-skill", nil)
	require.ErrorContains(t, err, "workspace root is empty")
	_, err = ResolveCustomProviderPackage(workspace, "sample-skill", nil)
	require.ErrorContains(t, err, "was not found in local or global skill roots")
}

func TestResolveCustomProviderPackageRejectsIdentityMismatchAndNonDirectory(t *testing.T) {
	workspace := t.TempDir()
	writeResolverSkill(t, filepath.Join(workspace, ".agents", "skills", "other-name"), resolverSkill)
	_, err := ResolveCustomProviderPackage(workspace, "other-name", nil)
	require.ErrorContains(t, err, `declares package id "sample-skill"`)

	skills := filepath.Join(workspace, ".codex", "skills")
	require.NoError(t, os.MkdirAll(skills, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skills, "flat-file"), []byte("x"), 0o644))
	_, err = ResolveCustomProviderPackage(workspace, "flat-file", nil)
	require.ErrorContains(t, err, "candidate is not a directory")
}

func TestPathPresentReportsStatFailures(t *testing.T) {
	present, err := pathPresent(filepath.Join(t.TempDir(), "missing"))
	require.NoError(t, err)
	require.False(t, present)

	file := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))
	_, err = pathPresent(filepath.Join(file, "child"))
	require.ErrorContains(t, err, "stat ")
}

func TestDefaultGlobalProviderRootsRequiresHome(t *testing.T) {
	require.Nil(t, DefaultGlobalProviderRoots(" "))
	require.Len(t, DefaultGlobalProviderRoots("/home/user"), 2)
}

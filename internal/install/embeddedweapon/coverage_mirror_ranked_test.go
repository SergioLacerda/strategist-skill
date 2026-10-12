package embeddedweapon

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/installtest"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopySkillPackageRejectsUnsafeEntries(t *testing.T) {
	installtest.RequireSymlinkSupport(t)
	t.Parallel()
	src := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(src, "nested"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "nested", "a.md"), []byte("a"), 0o644))
	target := t.TempDir()
	require.NoError(t, CopySkillPackage(src, target))
	assert.FileExists(t, filepath.Join(target, "nested", "a.md"))

	require.Error(t, CopySkillPackage(filepath.Join(src, "absent"), t.TempDir()))

	linked := t.TempDir()
	require.NoError(t, os.Symlink(src, filepath.Join(linked, "link")))
	require.ErrorContains(t, CopySkillPackage(linked, t.TempDir()), "symlink is not allowed")

	blocker := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocker, nil, 0o644))
	require.Error(t, MakeSkillDirectory(filepath.Join(blocker, "sub")))
	require.Error(t, CopySkillFile(filepath.Join(src, "absent"), filepath.Join(t.TempDir(), "x")))
	require.Error(t, CopySkillFile(filepath.Join(src, "nested", "a.md"), filepath.Join(blocker, "x")))
}

func TestWriteSkillMirrorFailures(t *testing.T) {
	t.Parallel()
	err := WriteSkillMirror(plugincatalog.PluginCatalog{}, plugincatalog.IngestedSkill{ID: "ghost"}, t.TempDir())
	require.NoError(t, err)

	blocker := filepath.Join(t.TempDir(), "root")
	require.NoError(t, os.WriteFile(blocker, nil, 0o644))
	catalog := plugincatalog.PluginCatalog{Providers: []plugincatalog.Provider{{ID: "x", Version: "1.0.0"}}}
	skill := plugincatalog.IngestedSkill{ID: "x"}
	skill.Package.Version = "1.0.0"
	err = WriteSkillMirror(catalog, skill, blocker)
	require.ErrorContains(t, err, "mkdir")

	root := t.TempDir()
	skill.Dir = filepath.Join(t.TempDir(), "absent")
	require.ErrorContains(t, WriteSkillMirror(catalog, skill, root), "copy package for x")
	require.NoError(t, writeSkillsAndLegacy(catalog, root))
}

func writeSkillsAndLegacy(catalog plugincatalog.PluginCatalog, root string) error {
	skill := plugincatalog.IngestedSkill{ID: "x"}
	skill.Package.Version = "1.0.0"
	if err := os.MkdirAll(filepath.Join(root, "skills", "x"), 0o755); err != nil {
		return err
	}
	return WriteSkillMirrors(catalog, []plugincatalog.IngestedSkill{skill}, root)
}

func TestRankedBindingCompilationFailures(t *testing.T) {
	t.Parallel()
	provider := plugincatalog.Provider{ID: "p", Version: "1.0.0", Roles: []string{"ranger"}, Ranked: true, CertificationDigest: "sha256:x", SupportedSlots: []string{"discovery"}}
	weapon := domain.CompiledWeapon{ID: "p", Version: "1.0.0"}
	role := domain.CompiledRole{ID: "ranger", Slot: "discovery"}

	_, err := CompileProviderRankedBindings(provider, nil, []domain.CompiledRole{role})
	require.ErrorContains(t, err, "is absent from compiled Weapons")
	_, err = CompileProviderRankedBindings(provider, []domain.CompiledWeapon{weapon}, nil)
	require.ErrorContains(t, err, "references unknown Role")

	_, err = BuildCompiledRankedBindings(plugincatalog.PluginCatalog{Providers: []plugincatalog.Provider{provider}}, nil, []domain.CompiledRole{role})
	require.Error(t, err)
	bindings, err := BuildCompiledRankedBindings(plugincatalog.PluginCatalog{Providers: []plugincatalog.Provider{provider, {ID: "skip"}}}, []domain.CompiledWeapon{weapon}, []domain.CompiledRole{role})
	require.NoError(t, err)
	require.Len(t, bindings, 1)
	assert.Equal(t, "discover", bindings[0].Entrypoint)

	assert.Equal(t, "refine", CompiledEntrypoint([]string{"refinement"}, "x"))
	assert.Equal(t, "execute", CompiledEntrypoint([]string{"execution"}, "x"))
	assert.Equal(t, "execute", CompiledEntrypoint(nil, "sniper"))
	assert.Equal(t, "invoke", CompiledEntrypoint(nil, "other"))
}

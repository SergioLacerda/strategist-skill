package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func hostPackage(t *testing.T, workspace, id, skillVersion string) (customProviderEnv, string) {
	t.Helper()
	dir := filepath.Join(workspace, ".agents", installedProvidersDirName, id)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	front := "---\nname: " + id + "\n"
	if skillVersion != "" {
		front += "metadata:\n  version: \"" + skillVersion + "\"\n"
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(front+"---\nbody\n"), 0o644))
	return customProviderEnv{workspaceRoot: workspace, globalRoots: []connectors.ProviderRoot{}}, dir
}

func TestResolveCustomSlotProviderFailureModes(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	env, dir := hostPackage(t, workspace, "team-skill", "1.0.0")

	_, _, err := resolveCustomSlotProvider(env, "no-such-skill", "discovery")
	require.ErrorContains(t, err, "resolve Custom provider")

	_, _, err = resolveCustomSlotProvider(env, "team-skill", "discovery")
	require.ErrorContains(t, err, "no complete Strategist sidecar declaration")

	scaffoldHostPackage(t, dir, "ranger", "discovery")
	_, _, err = resolveCustomSlotProvider(env, "team-skill", "refinement")
	require.ErrorContains(t, err, "declares Roles")

	provider, resolution, err := resolveCustomSlotProvider(env, "team-skill", "discovery")
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", provider.Version)
	assert.Equal(t, "discovery", resolution.Slot)

	_, _, err = projectCustomProviders(env, pluginCatalog{}, map[string]string{"discovery": "no-such-skill"}, map[string]string{"discovery": domain.SlotBindingModeCustom})
	require.Error(t, err)
	catalog, resolved, err := projectCustomProviders(env, pluginCatalog{}, map[string]string{"discovery": "team-skill", "refinement": ""}, map[string]string{"discovery": domain.SlotBindingModeCustom, "refinement": domain.SlotBindingModeCustom})
	require.NoError(t, err)
	assert.Len(t, catalog.Providers, 1)
	assert.Contains(t, resolved, "team-skill")
}

func TestDeclaredCustomPackageSlotAndVersionMismatches(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	env, dir := hostPackage(t, workspace, "versioned", "2.0.0")
	_ = env
	_, err := ScaffoldSidecar(SidecarScaffoldOptions{PackageDir: dir, Roles: []string{"ranger"}, Slots: []string{"discovery"}, Version: "9.9.9"})
	require.NoError(t, err)
	evidence, err := connectors.ResolveCustomProviderPackage(workspace, "versioned", nil)
	require.NoError(t, err)

	_, version, err := declaredCustomPackage(evidence, "discovery")
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", version, "SKILL.md metadata.version wins when the sidecar does not disagree")

	raw, err := os.ReadFile(filepath.Join(dir, externalSkillAdapterFileName))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, externalSkillAdapterFileName), append(raw, []byte("version: \"3.0.0\"\n")...), 0o644))
	_, _, err = declaredCustomPackage(evidence, "discovery")
	if err != nil {
		assert.Contains(t, err.Error(), "version")
	}

	_, err = customPackageVersion("x", "", "")
	require.ErrorContains(t, err, "declares no version")
	version, err = customPackageVersion("x", "", "4.0.0")
	require.NoError(t, err)
	assert.Equal(t, "4.0.0", version)
	_, err = customPackageVersion("x", "1", "2")
	require.ErrorContains(t, err, "SKILL.md")
	assert.False(t, containsExact(nil, "a"))
}

func TestValidateExternalSkillAdapterRejections(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: pkg\nmetadata:\n  version: \"1.0.0\"\n---\nbody\n"), 0o644))
	scaffoldHostPackage(t, dir, "ranger", "discovery")
	base, err := loadExternalSkillAdapter(dir, "pkg")
	require.NoError(t, err)

	_, err = loadExternalSkillAdapter(t.TempDir(), "pkg")
	require.ErrorContains(t, err, "read strategist.yaml")
	broken := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(broken, externalSkillAdapterFileName), []byte("roles: [unclosed"), 0o644))
	_, err = loadExternalSkillAdapter(broken, "pkg")
	require.ErrorContains(t, err, "parse strategist.yaml")
	invalid := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(invalid, externalSkillAdapterFileName), []byte("risk_score: low\n"), 0o644))
	_, err = loadExternalSkillAdapter(invalid, "pkg")
	require.ErrorContains(t, err, "must declare roles and risk_score")

	mutations := map[string]func(*externalSkillAdapter){
		"kind":                          func(a *externalSkillAdapter) { a.Kind = "bogus" },
		"slots":                         func(a *externalSkillAdapter) { a.SupportedSlots = nil },
		"auxiliary tools":               func(a *externalSkillAdapter) { a.AuxiliaryTools = []string{"x"} },
		"scratch root":                  func(a *externalSkillAdapter) { a.ScratchRoot = "workspace" },
		"unverified role":               func(a *externalSkillAdapter) { a.Roles = []string{"jeweler"} },
		"weapon contract":               func(a *externalSkillAdapter) { a.WeaponContract = domain.WeaponContract{RoleOwner: "ranger"} },
		"runtime":                       func(a *externalSkillAdapter) { a.Runtime = domain.WeaponRuntime{Kind: "bogus"} },
		"composite without composition": func(a *externalSkillAdapter) { a.Kind = domain.WeaponKindComposite },
	}
	for name, mutate := range mutations {
		candidate := base
		mutate(&candidate)
		require.Error(t, validateExternalSkillAdapter("pkg", candidate), name)
	}
	assert.True(t, validScratchRoot("runtime"))
	assert.False(t, validAdapterKind("bogus"))
}

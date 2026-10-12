package customprovider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/SergioLacerda/strategist-skill/internal/install/sidecar"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func hostPackage(t *testing.T, workspace, id, skillVersion string) (Env, string) {
	t.Helper()
	dir := filepath.Join(workspace, ".agents", names.SkillsDirName, id)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	front := "---\nname: " + id + "\n"
	if skillVersion != "" {
		front += "metadata:\n  version: \"" + skillVersion + "\"\n"
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(front+"---\nbody\n"), 0o644))
	return Env{workspaceRoot: workspace, globalRoots: []connectors.ProviderRoot{}}, dir
}

func TestResolveCustomSlotProviderFailureModes(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	env, dir := hostPackage(t, workspace, "team-skill", "1.0.0")

	_, _, err := ResolveCustomSlotProvider(env, "no-such-skill", "discovery")
	require.ErrorContains(t, err, "resolve Custom provider")

	_, _, err = ResolveCustomSlotProvider(env, "team-skill", "discovery")
	require.ErrorContains(t, err, "no complete Strategist sidecar declaration")

	scaffoldCustomHostPackage(t, dir, "ranger", "discovery")
	_, _, err = ResolveCustomSlotProvider(env, "team-skill", "refinement")
	require.ErrorContains(t, err, "declares Roles")

	provider, resolution, err := ResolveCustomSlotProvider(env, "team-skill", "discovery")
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", provider.Version)
	assert.Equal(t, "discovery", resolution.Slot)

	_, _, err = ProjectCustomProviders(env, plugincatalog.PluginCatalog{}, map[string]string{"discovery": "no-such-skill"}, map[string]string{"discovery": domain.SlotBindingModeCustom})
	require.Error(t, err)
	catalog, resolved, err := ProjectCustomProviders(env, plugincatalog.PluginCatalog{}, map[string]string{"discovery": "team-skill", "refinement": ""}, map[string]string{"discovery": domain.SlotBindingModeCustom, "refinement": domain.SlotBindingModeCustom})
	require.NoError(t, err)
	assert.Len(t, catalog.Providers, 1)
	assert.Contains(t, resolved, "team-skill")
}

func TestDeclaredCustomPackageSlotAndVersionMismatches(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	env, dir := hostPackage(t, workspace, "versioned", "2.0.0")
	_ = env
	_, err := sidecar.ScaffoldSidecar(sidecar.ScaffoldOptions{PackageDir: dir, Roles: []string{"ranger"}, Slots: []string{"discovery"}, Version: "9.9.9"})
	require.NoError(t, err)
	evidence, err := connectors.ResolveCustomProviderPackage(workspace, "versioned", nil)
	require.NoError(t, err)

	_, version, err := DeclaredCustomPackage(evidence, "discovery")
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", version, "SKILL.md metadata.version wins when the sidecar does not disagree")

	raw, err := os.ReadFile(filepath.Join(dir, plugincatalog.ExternalSkillAdapterFileName))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, plugincatalog.ExternalSkillAdapterFileName), append(raw, []byte("version: \"3.0.0\"\n")...), 0o644))
	_, _, err = DeclaredCustomPackage(evidence, "discovery")
	if err != nil {
		assert.Contains(t, err.Error(), "version")
	}

	_, err = CustomPackageVersion("x", "", "")
	require.ErrorContains(t, err, "declares no version")
	version, err = CustomPackageVersion("x", "", "4.0.0")
	require.NoError(t, err)
	assert.Equal(t, "4.0.0", version)
	_, err = CustomPackageVersion("x", "1", "2")
	require.ErrorContains(t, err, "SKILL.md")
	assert.False(t, ContainsExact(nil, "a"))
}

func TestValidateExternalSkillAdapterRejections(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: pkg\nmetadata:\n  version: \"1.0.0\"\n---\nbody\n"), 0o644))
	scaffoldCustomHostPackage(t, dir, "ranger", "discovery")
	base, err := plugincatalog.LoadExternalSkillAdapter(dir, "pkg")
	require.NoError(t, err)

	_, err = plugincatalog.LoadExternalSkillAdapter(t.TempDir(), "pkg")
	require.ErrorContains(t, err, "read strategist.yaml")
	broken := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(broken, plugincatalog.ExternalSkillAdapterFileName), []byte("roles: [unclosed"), 0o644))
	_, err = plugincatalog.LoadExternalSkillAdapter(broken, "pkg")
	require.ErrorContains(t, err, "parse strategist.yaml")
	invalid := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(invalid, plugincatalog.ExternalSkillAdapterFileName), []byte("risk_score: low\n"), 0o644))
	_, err = plugincatalog.LoadExternalSkillAdapter(invalid, "pkg")
	require.ErrorContains(t, err, "must declare roles and risk_score")

	mutations := map[string]func(*plugincatalog.ExternalSkillAdapter){
		"kind":            func(a *plugincatalog.ExternalSkillAdapter) { a.Kind = "bogus" },
		"slots":           func(a *plugincatalog.ExternalSkillAdapter) { a.SupportedSlots = nil },
		"auxiliary tools": func(a *plugincatalog.ExternalSkillAdapter) { a.AuxiliaryTools = []string{"x"} },
		"scratch root":    func(a *plugincatalog.ExternalSkillAdapter) { a.ScratchRoot = "workspace" },
		"unverified role": func(a *plugincatalog.ExternalSkillAdapter) { a.Roles = []string{"jewelcrafter"} },
		"weapon contract": func(a *plugincatalog.ExternalSkillAdapter) {
			a.WeaponContract = domain.WeaponContract{RoleOwner: "ranger"}
		},
		"runtime":                       func(a *plugincatalog.ExternalSkillAdapter) { a.Runtime = domain.WeaponRuntime{Kind: "bogus"} },
		"composite without composition": func(a *plugincatalog.ExternalSkillAdapter) { a.Kind = domain.WeaponKindComposite },
	}
	for name, mutate := range mutations {
		candidate := base
		mutate(&candidate)
		require.Error(t, plugincatalog.ValidateExternalSkillAdapter("pkg", candidate), name)
	}
	assert.True(t, plugincatalog.ValidScratchRoot("runtime"))
	assert.False(t, plugincatalog.ValidAdapterKind("bogus"))
}

// scaffoldCustomHostPackage writes the Strategist sidecar a wizard-selected host
// Weapon must declare, through the real generator, never inferred by the wizard.
func scaffoldCustomHostPackage(t *testing.T, dir, role, slot string) {
	t.Helper()
	_, err := sidecar.ScaffoldSidecar(sidecar.ScaffoldOptions{PackageDir: dir, Roles: []string{role}, Slots: []string{slot}})
	require.NoError(t, err)
}

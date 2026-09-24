package domain_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRoleRegistryDescribesTodaysRoles(t *testing.T) {
	reg := domain.DefaultRoleRegistry()
	assert.Equal(t, []string{"scout", "ranger", "archivist", "sniper"}, reg.IDs(), "ordered by phase")

	cases := map[string]struct {
		slot          string
		phase         int
		origin        domain.RoleOrigin
		extensibility domain.RoleExtensibility
		pluggable     bool
	}{
		"scout":     {"", 0, domain.RoleOriginNative, domain.RoleExtensibilityFixed, false},
		"ranger":    {"discovery", 1, domain.RoleOriginNative, domain.RoleExtensibilityPluggable, true},
		"archivist": {"refinement", 2, domain.RoleOriginNative, domain.RoleExtensibilityPluggable, true},
		"sniper":    {"execution", 4, domain.RoleOriginNative, domain.RoleExtensibilityFixed, false},
	}
	for id, want := range cases {
		role, ok := reg.Get(id)
		require.True(t, ok, id)
		assert.Equal(t, want.slot, role.Slot, id)
		assert.Equal(t, want.phase, role.Phase, id)
		assert.Equal(t, want.origin, role.Origin, id)
		assert.Equal(t, want.extensibility, role.Extensibility, id)
		assert.Equal(t, want.pluggable, role.Pluggable, id)
	}
	assert.Equal(t, "schemas/handoff-ranger-to-archivist.schema.yaml", reg.HandoffSchemaOf("ranger"))
	assert.Equal(t, "schemas/handoff-archivist-to-sniper.schema.yaml", reg.HandoffSchemaOf("archivist"))
	assert.Empty(t, reg.HandoffSchemaOf("sniper"), "terminal role hands nothing downstream")
	for _, unverified := range []string{"pathfinder", "cartographer", "jeweler", "jewelcrafter"} {
		assert.False(t, reg.Has(unverified), "%s must not be activated by the current taxonomy", unverified)
	}
}

func TestRoleRegistryLookups(t *testing.T) {
	reg := domain.DefaultRoleRegistry()
	assert.True(t, reg.Has("Ranger"), "lookups are case-insensitive")
	assert.False(t, reg.Has("gate"), "the gate is a pipeline step, not a role")
	assert.False(t, reg.Has("response_critic"), "sub-routines are not roles")
	assert.Equal(t, "ranger", reg.PolicyRole("ranger"), "leveling key defaults to the role id")
	assert.Equal(t, "unknown", reg.PolicyRole("unknown"))

	role, ok := reg.RoleForSlot("execution")
	require.True(t, ok)
	assert.Equal(t, "sniper", role.ID)
	_, ok = reg.RoleForSlot("nope")
	assert.False(t, ok)
}

func TestRoleRegistryExposesIndependentInitiativeHooks(t *testing.T) {
	reg := domain.DefaultRoleRegistry()
	hooks, ok := reg.InitiativeHooksOf("ranger")
	require.True(t, ok)
	assert.Equal(t, "resolve_advice", hooks.OnStart)
	assert.Equal(t, "emit_initiative_result", hooks.OnResult)
	assert.Contains(t, hooks.Preserve, "advice_id")
	assert.Contains(t, hooks.Preserve, "alignment")

	_, ok = reg.InitiativeHooksOf("unknown")
	assert.False(t, ok)
}

func TestRoleRegistryPhaseCounterIncludesGateAndDerivesTotal(t *testing.T) {
	reg := domain.DefaultRoleRegistry()
	for id, want := range map[string]int{"scout": 0, "ranger": 1, "archivist": 2, "gate": 3, "sniper": 4} {
		phase, ok := reg.PhaseOf(id)
		require.True(t, ok, id)
		assert.Equal(t, want, phase, id)
	}
	_, ok := reg.PhaseOf("transport")
	assert.False(t, ok)
	assert.Equal(t, 4, reg.PhaseTotal())
}

func TestRoleRegistryTotalChangesWhenARoleIsAdded(t *testing.T) {
	roles := domain.DefaultRoleRegistry().Roles()
	roles = append(roles, domain.Role{ID: "auditor", Phase: 5})
	reg, err := domain.NewRoleRegistry(roles)
	require.NoError(t, err)
	assert.Equal(t, 5, reg.PhaseTotal(), "no code edit is needed to change the total")
	assert.Equal(t, []string{"scout", "ranger", "archivist", "sniper", "auditor"}, reg.IDs())
}

func TestNewRoleRegistryRejectsInvalidDefinitions(t *testing.T) {
	cases := map[string][]domain.Role{
		"empty id":               {{ID: " "}},
		"duplicate id":           {{ID: "a"}, {ID: "A"}},
		"duplicate phase":        {{ID: "a", Phase: 1}, {ID: "b", Phase: 1}},
		"unknown slot":           {{ID: "a", Slot: "planning"}},
		"pluggable without slot": {{ID: "a", Pluggable: true}},
		"negative phase":         {{ID: "a", Phase: -1}},
	}
	for name, roles := range cases {
		_, err := domain.NewRoleRegistry(roles)
		require.Error(t, err, name)
	}
}

func TestRoleConfigSlotlessRoleMustDeclareNotPluggable(t *testing.T) {
	no, yes := false, true
	require.NoError(t, domain.RoleConfig{Role: "scout", Pluggable: &no}.Validate(), "explicitly non-pluggable, no slot")
	require.Error(t, domain.RoleConfig{Role: "scout"}.Validate(), "legacy files still need a slot")
	require.Error(t, domain.RoleConfig{Role: "scout", Pluggable: &yes}.Validate(), "a pluggable role needs a slot")
	require.NoError(t, domain.RoleConfig{Role: "ranger", Slot: "discovery"}.Validate())
	require.NoError(t, domain.RoleConfig{Role: "scout", Extensibility: domain.RoleExtensibilityFixed}.Validate())
	require.Error(t, domain.RoleConfig{Role: "scout", Extensibility: domain.RoleExtensibilityPluggable}.Validate())
}

func TestRoleTaxonomyKeepsOriginIndependentFromExtensibility(t *testing.T) {
	role := domain.Role{ID: "ranger", Origin: domain.RoleOriginExternal, Extensibility: domain.RoleExtensibilityFixed, Slot: "discovery"}
	reg, err := domain.NewRoleRegistry([]domain.Role{role})
	require.NoError(t, err)
	got, ok := reg.Get("ranger")
	require.True(t, ok)
	assert.Equal(t, domain.RoleOriginExternal, got.Origin)
	assert.Equal(t, domain.RoleExtensibilityFixed, got.Extensibility)
	assert.False(t, got.Pluggable)
}

func TestRoleRegistryRejectsUnknownTaxonomyValues(t *testing.T) {
	_, err := domain.NewRoleRegistry([]domain.Role{{ID: "ranger", Origin: "vendor", Slot: "discovery"}})
	require.ErrorContains(t, err, "role origin")
	_, err = domain.NewRoleRegistry([]domain.Role{{ID: "ranger", Extensibility: "conditional", Slot: "discovery"}})
	require.ErrorContains(t, err, "role extensibility")
}

func TestRoleRegistryRejectsUnverifiedRoleActivation(t *testing.T) {
	for _, roleID := range []string{"Pathfinder", "Cartographer", "Jeweler", "Jewelcrafter"} {
		t.Run(roleID, func(t *testing.T) {
			_, err := domain.NewRoleRegistry([]domain.Role{{ID: roleID}})
			require.ErrorContains(t, err, "not approved for activation")
		})
	}
}

func writeRoleFile(t *testing.T, dir, name, body string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600))
}

func TestLoadRoleRegistryOverlaysFilesOnBuiltIns(t *testing.T) {
	dir := t.TempDir()
	writeRoleFile(t, dir, "default.yaml", "discovery: ranger\nrefinement: archivist\nexecution: sniper\n")
	writeRoleFile(t, dir, "ranger.yaml", "role: ranger\nslot: discovery\nphase: 1\npluggable: true\nleveling: scout\nhandoff_schema: schemas/custom.yaml\ncanonical:\n  - x\n")
	writeRoleFile(t, dir, "auditor.yaml", "role: auditor\nphase: 5\npluggable: false\n")

	reg, err := domain.LoadRoleRegistry(dir)
	require.NoError(t, err)
	assert.Equal(t, "scout", reg.PolicyRole("ranger"), "an explicit leveling key is honored")
	assert.Equal(t, "schemas/custom.yaml", reg.HandoffSchemaOf("ranger"))
	hooks, ok := reg.InitiativeHooksOf("ranger")
	require.True(t, ok)
	assert.Equal(t, "resolve_advice", hooks.OnStart, "legacy role overrides keep INITIATIVE enabled")
	assert.True(t, reg.Has("auditor"), "a new role file adds a role")
	assert.True(t, reg.Has("sniper"), "roles without files keep the built-in definition")
	assert.Equal(t, 5, reg.PhaseTotal())
}

func TestLoadRoleRegistryMissingDirUsesBuiltIns(t *testing.T) {
	reg, err := domain.LoadRoleRegistry(filepath.Join(t.TempDir(), "absent"))
	require.NoError(t, err)
	assert.Equal(t, domain.DefaultRoleRegistry().IDs(), reg.IDs())
}

func TestLoadRoleRegistryRejectsMalformedRoleFile(t *testing.T) {
	dir := t.TempDir()
	writeRoleFile(t, dir, "broken.yaml", ":\n - [nope")
	_, err := domain.LoadRoleRegistry(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "broken.yaml")
}

func TestLevelingRolesFollowTheRegistry(t *testing.T) {
	assert.Equal(t, domain.DefaultRoleRegistry().IDs(), domain.LevelingRoleIDs())
}

func TestEveryRoleDeclaresTheLevelStartHook(t *testing.T) {
	reg := domain.DefaultRoleRegistry()
	for _, id := range reg.IDs() {
		commands := reg.StartCommands(id, "m-42")
		require.NotEmpty(t, commands, "%s must resolve its level at start", id)
		assert.Equal(t, []string{"strategist leveling label --role " + id + " --mission m-42 --host-model <your-model> --host-effort <your-effort>"}, commands, id)
	}
	assert.Empty(t, reg.StartCommands("transport", "m-42"), "an unregistered role has no hook")
}

func TestStartCommandsComeFromTheRoleDefinition(t *testing.T) {
	dir := t.TempDir()
	writeRoleFile(t, dir, "ranger.yaml", "role: ranger\nslot: discovery\nphase: 1\npluggable: true\non_start:\n  - strategist leveling label --role {role} --mission {mission_id} --run 2\n")
	reg, err := domain.LoadRoleRegistry(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{"strategist leveling label --role ranger --mission m9 --run 2"}, reg.StartCommands("ranger", "m9"))
}

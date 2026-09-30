package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// fixtureRegistry is a valid compiled registry with the permanent pairings
// (brainstorming<->ranger, openspec-propose<->archivist) plus the native
// sniper role, which the registry-derived roster must not scan as a Weapon.
func fixtureRegistry() domain.CompiledRegistry {
	bridge := domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded, ExecutionMode: domain.WeaponExecutionModePromptBridge}
	code := domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded, ExecutionMode: domain.WeaponExecutionModeCode}
	weapon := func(id string, runtime domain.WeaponRuntime) domain.CompiledWeapon {
		source := ""
		if runtime.ExecutionMode == domain.WeaponExecutionModePromptBridge {
			source = "sha256:source-" + id
		}
		return domain.CompiledWeapon{ID: id, Version: "1.0.0", Digest: "sha256:weapon-" + id, SourceDigest: source, Origin: domain.WeaponOriginEmbedded, Runtime: runtime}
	}
	binding := func(role, slot, weaponID string, runtime domain.WeaponRuntime) domain.CompiledRankedBinding {
		w := weapon(weaponID, runtime)
		return domain.CompiledRankedBinding{
			Role: role, Slot: slot, WeaponID: weaponID, WeaponVersion: w.Version, WeaponDigest: w.Digest, RoleDigest: "sha256:role-" + role,
			BindingDigest: "sha256:binding-" + role, CertificationDigest: "sha256:cert-" + role, SourceDigest: w.SourceDigest,
			ExecutionMode: runtime.ExecutionMode, ConnectorID: "connector", Runtime: runtime, Entrypoint: "invoke", Generation: 1, Status: "active",
		}
	}
	return domain.CompiledRegistry{
		SchemaVersion: domain.CompiledRegistrySchemaVersion,
		Weapons: []domain.CompiledWeapon{
			weapon("brainstorming", bridge), weapon("openspec-propose", bridge), weapon("sniper", code),
		},
		Roles: []domain.CompiledRole{
			{ID: "ranger", Slot: "discovery", ContractDigest: "sha256:role-ranger"},
			{ID: "archivist", Slot: "refinement", ContractDigest: "sha256:role-archivist"},
			{ID: "sniper", Slot: "execution", ContractDigest: "sha256:role-sniper"},
		},
		RankedBindings: []domain.CompiledRankedBinding{
			binding("ranger", "discovery", "brainstorming", bridge),
			binding("archivist", "refinement", "openspec-propose", bridge),
			binding("sniper", "execution", "sniper", code),
		},
	}
}

// registrySectionsYAML renders the compiled registry sections that the
// generated plugins/catalog.yaml carries next to its providers.
func registrySectionsYAML(t *testing.T, registry domain.CompiledRegistry) string {
	t.Helper()
	raw, err := yaml.Marshal(struct {
		Weapons        []domain.CompiledWeapon        `yaml:"weapons"`
		Roles          []domain.CompiledRole          `yaml:"roles"`
		RankedBindings []domain.CompiledRankedBinding `yaml:"ranked_bindings"`
	}{registry.Weapons, registry.Roles, registry.RankedBindings})
	require.NoError(t, err)
	return string(raw)
}

// writeRegistryCatalog writes a catalog holding only the compiled registry.
func writeRegistryCatalog(t *testing.T, root string, registry domain.CompiledRegistry) {
	t.Helper()
	body := "schema_version: strategist-plugin-catalog/v2\n" + registrySectionsYAML(t, registry)
	require.NoError(t, os.MkdirAll(filepath.Join(root, "plugins"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins", "catalog.yaml"), []byte(body), 0o644))
}

// appendRegistrySections adds the compiled registry sections to an existing
// fixture catalog, as the generated plugins/catalog.yaml carries them.
func appendRegistrySections(t *testing.T, root string, registry domain.CompiledRegistry) {
	t.Helper()
	f, err := os.OpenFile(filepath.Join(root, "plugins", "catalog.yaml"), os.O_APPEND|os.O_WRONLY, 0o644)
	require.NoError(t, err)
	defer func() { require.NoError(t, f.Close()) }()
	_, err = f.WriteString(registrySectionsYAML(t, registry))
	require.NoError(t, err)
}

// rankedLockFields renders the lock fields a Ranked binding copies from the
// compiled registry, indented for a bindings entry.
func rankedLockFields(t *testing.T, role string) string {
	t.Helper()
	registry := fixtureRegistry()
	for _, binding := range registry.RankedBindings {
		if binding.Role != role {
			continue
		}
		return "    role: " + role + "\n" +
			"    weapon_version: " + binding.WeaponVersion + "\n" +
			"    weapon_digest: " + binding.WeaponDigest + "\n" +
			"    source_digest: " + binding.SourceDigest + "\n" +
			"    binding_digest: " + binding.BindingDigest + "\n" +
			"    execution_mode: " + binding.ExecutionMode + "\n" +
			"    runtime_kind: " + binding.Runtime.Kind + "\n" +
			"    connector_id: " + binding.ConnectorID + "\n" +
			"    entrypoint: " + binding.Entrypoint + "\n" +
			"    certification_digest: " + binding.CertificationDigest + "\n"
	}
	t.Fatalf("no fixture Ranked binding for role %s", role)
	return ""
}

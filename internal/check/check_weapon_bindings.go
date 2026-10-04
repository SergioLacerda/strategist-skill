package check

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/weapon"
	"gopkg.in/yaml.v3"
)

// weaponBinding is the verification result for one embedded skill (weapon)
// that declares a canonical_role. This check runs unconditionally over every
// skill under
// <root>/skills/ that declares specialization_taxonomy.canonical_role,
// independent of what active.yaml currently configures for any slot.
type weaponBinding struct {
	SkillID       string
	CanonicalRole string
	PayloadDir    string // skills/ directory name of a catalogued Weapon (id@version)
	Slot          string // resolved from roles/<CanonicalRole>.yaml's own slot field; empty when Reason is set before the role file is read
	OK            bool
	Reason        string // non-empty only when OK is false
}

// verifyEmbeddedWeaponBindings scans the catalog roster. For each embedded
// entry declaring a canonical_role, it verifies the claimed pairing is
// structurally intact end to end: the target role file
// (<root>/roles/<canonical_role>.yaml) exists and is a valid RoleConfig, and
// roles/default.yaml maps that role's own declared slot back to the same
// role. Skills with no canonical_role are not embedded weapons and are
// skipped, not reported — this check is scoped to the embedded-weapon
// roster (DEC-001: brainstorming↔ranger, openspec-propose↔archivist),
// never to every installed skill.
func verifyEmbeddedWeaponBindings(root string) ([]weaponBinding, error) {
	catalog, err := weapon.ListCatalogWeaponFacts(root)
	if err != nil {
		return nil, fmt.Errorf("weapon bindings: %w", err)
	}
	// A missing/invalid roles/default.yaml is surfaced per-binding below
	// (each binding needing it fails with an explicit reason), not as a
	// function-level error so each expected binding receives an explicit reason.
	roleSlotMap, roleSlotMapErr := loadRoleSlotMap(root)

	// The catalog is the roster authority (DEC-009): every embedded entry that
	// declares a canonical role is verified without reading any compatibility
	// view. verifyEmbeddedWeaponRoster closes the converse direction by ensuring
	// every permanent pairing has a payload.
	bindings, _ := catalogWeaponBindings(root, catalog, roleSlotMap, roleSlotMapErr)
	bindings = append(bindings, verifyEmbeddedWeaponRoster(root, bindings)...)
	return bindings, nil
}

func verifyOneWeaponBinding(root, skillID, canonicalRole string, weaponContract domain.WeaponContract, roleSlotMap domain.RoleSlotMap, roleSlotMapErr error) weaponBinding {
	b := weaponBinding{SkillID: skillID, CanonicalRole: canonicalRole}

	rolePath := filepath.Join(root, "roles", canonicalRole+".yaml")
	roleRaw, err := os.ReadFile(rolePath) //nolint:gosec // G304: path derived from the runtime roles directory
	if err != nil {
		b.Reason = fmt.Sprintf("canonical_role %q: role file missing (%s)", canonicalRole, rolePath)
		return b
	}
	var roleCfg domain.RoleConfig
	if yamlErr := yaml.Unmarshal(roleRaw, &roleCfg); yamlErr != nil {
		b.Reason = fmt.Sprintf("canonical_role %q: role file malformed YAML: %v", canonicalRole, yamlErr)
		return b
	}
	if valErr := roleCfg.Validate(); valErr != nil {
		b.Reason = fmt.Sprintf("canonical_role %q: role config invalid: %v", canonicalRole, valErr)
		return b
	}
	b.Slot = roleCfg.Slot

	if roleSlotMapErr != nil {
		b.Reason = fmt.Sprintf("roles/default.yaml unreadable: %v", roleSlotMapErr)
		return b
	}
	if roleSlotMap[roleCfg.Slot] != canonicalRole {
		b.Reason = fmt.Sprintf("roles/default.yaml maps slot %q to %q, not %q", roleCfg.Slot, roleSlotMap[roleCfg.Slot], canonicalRole)
		return b
	}
	if err := validateWeaponBoundary(roleCfg.Slot, canonicalRole, weaponContract); err != nil {
		b.Reason = err.Error()
		return b
	}

	b.OK = true
	return b
}

// catalogWeaponBindings verifies every embedded catalog entry that declares a
// canonical role.
func catalogWeaponBindings(root string, catalog []domain.WeaponFacts, roleSlotMap domain.RoleSlotMap, roleSlotMapErr error) ([]weaponBinding, map[string]bool) {
	var bindings []weaponBinding
	covered := map[string]bool{}
	for _, facts := range catalog {
		if facts.CompatibilitySource != "embedded" || facts.CanonicalRole == "" {
			continue
		}
		payloadDir := domain.WeaponPayloadDirName(facts.ID, facts.Version)
		covered[facts.ID] = true
		covered[payloadDir] = true
		b := verifyOneWeaponBinding(root, facts.ID, facts.CanonicalRole, facts.WeaponContract, roleSlotMap, roleSlotMapErr)
		b.PayloadDir = payloadDir
		if b.OK {
			b = requireSkillPayload(root, b)
		}
		bindings = append(bindings, b)
	}
	return bindings, covered
}

// requireSkillPayload is the roster's separate payload check: an embedded Weapon
// must ship its skills/<id>@<version>/SKILL.md.
func requireSkillPayload(root string, b weaponBinding) weaponBinding {
	path := filepath.Join(root, "skills", b.PayloadDir, "SKILL.md")
	if info, err := os.Stat(path); err != nil || info.Size() == 0 {
		b.OK = false
		b.Reason = fmt.Sprintf("embedded weapon payload missing or empty: %s", path)
	}
	return b
}

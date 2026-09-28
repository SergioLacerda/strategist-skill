package check

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
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
	Slot          string // resolved from roles/<CanonicalRole>.yaml's own slot field; empty when Reason is set before the role file is read
	OK            bool
	Reason        string // non-empty only when OK is false
}

// skillTaxonomy accepts canonical_role in either shape a skill.yaml may
// carry it in: top-level (the internal/embed/defaults/skills/<id>/skill.yaml
// authoring convention brainstorming and openspec-explore already use) or
// nested under specialization_taxonomy (the shape a compiled runtime
// instance's .strategist/skills/<id>/skill.yaml may carry — the two are not
// currently guaranteed identical by any generator this check depends on, so
// it reads whichever is present rather than assuming one).
type skillTaxonomy struct {
	CanonicalRole          string                `yaml:"canonical_role"`
	Roles                  []string              `yaml:"roles"`
	WeaponContract         domain.WeaponContract `yaml:"weapon_contract"`
	SpecializationTaxonomy struct {
		CanonicalRole string `yaml:"canonical_role"`
	} `yaml:"specialization_taxonomy"`
}

func (t skillTaxonomy) canonicalRole() string {
	if t.CanonicalRole != "" {
		return t.CanonicalRole
	}
	return t.SpecializationTaxonomy.CanonicalRole
}

func (t skillTaxonomy) roles() []string {
	if len(t.Roles) > 0 {
		return t.Roles
	}
	if role := t.canonicalRole(); role != "" {
		return []string{role}
	}
	return nil
}

// verifyEmbeddedWeaponBindings scans every skill manifest under
// <root>/skills/*/skill.yaml. For each one declaring
// specialization_taxonomy.canonical_role, it verifies the claimed pairing is
// structurally intact end to end: the target role file
// (<root>/roles/<canonical_role>.yaml) exists and is a valid RoleConfig, and
// roles/default.yaml maps that role's own declared slot back to the same
// role. Skills with no canonical_role are not embedded weapons and are
// skipped, not reported — this check is scoped to the embedded-weapon
// roster (DEC-001: brainstorming↔ranger, openspec-propose↔archivist),
// never to every installed skill.
func verifyEmbeddedWeaponBindings(root string) ([]weaponBinding, error) {
	catalog, err := domain.ListCatalogWeaponFacts(root)
	if err != nil {
		return nil, fmt.Errorf("weapon bindings: %w", err)
	}
	skillsDir := filepath.Join(root, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("weapon bindings: read %s: %w", skillsDir, err)
	}

	// A missing/invalid roles/default.yaml is surfaced per-binding below
	// (each binding needing it fails with an explicit reason), not as a
	// function-level error so each expected binding receives an explicit reason.
	roleSlotMap, roleSlotMapErr := loadRoleSlotMap(root)

	// The catalog is the roster authority (DEC-009): every embedded entry that
	// declares a canonical role is verified, with no compat view needed. A view
	// directory the catalog does not list is still scanned during the transition.
	bindings, cataloged := catalogWeaponBindings(root, catalog, roleSlotMap, roleSlotMapErr)
	bindings = append(bindings, scanUncatalogedViews(root, skillsDir, entries, cataloged, roleSlotMap, roleSlotMapErr)...)
	// The scan above is reactive (skill declares a role -> is the claim
	// valid?). It never asks the converse question a role-focused reading of
	// DEC-003 requires: does each permanent embedded-weapon pairing actually
	// exist at all? A pairing whose skill directory is entirely absent
	// produces zero rows from the scan above — silence, not a reported
	// failure. verifyEmbeddedWeaponRoster closes that direction.
	bindings = append(bindings, verifyEmbeddedWeaponRoster(bindings)...)
	return bindings, nil
}

func scanWeaponEntry(root, skillsDir string, entry os.DirEntry, roleSlotMap domain.RoleSlotMap, roleSlotMapErr error) (weaponBinding, bool) {
	if !entry.IsDir() {
		return weaponBinding{}, false
	}
	skillID := entry.Name()
	raw, err := os.ReadFile(filepath.Join(skillsDir, skillID, "skill.yaml")) //nolint:gosec // G304: path derived from the runtime skills directory
	if err != nil {
		return weaponBinding{}, false
	}
	var taxonomy skillTaxonomy
	if err := yaml.Unmarshal(raw, &taxonomy); err != nil {
		return weaponBinding{SkillID: skillID, Reason: fmt.Sprintf("skill.yaml invalid: %v", err)}, true
	}
	canonicalRole := taxonomy.canonicalRole()
	if canonicalRole == "" {
		return weaponBinding{}, false
	}
	return verifyOneWeaponBinding(root, skillID, canonicalRole, taxonomy.WeaponContract, roleSlotMap, roleSlotMapErr), true
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
// canonical role, and returns the ids it covered so the transitional view scan
// skips them.
func catalogWeaponBindings(root string, catalog []domain.WeaponFacts, roleSlotMap domain.RoleSlotMap, roleSlotMapErr error) ([]weaponBinding, map[string]bool) {
	var bindings []weaponBinding
	covered := map[string]bool{}
	for _, facts := range catalog {
		if facts.CompatibilitySource != "embedded" || facts.CanonicalRole == "" {
			continue
		}
		covered[facts.ID] = true
		b := verifyOneWeaponBinding(root, facts.ID, facts.CanonicalRole, facts.WeaponContract, roleSlotMap, roleSlotMapErr)
		if b.OK {
			b = requireSkillPayload(root, b)
		}
		bindings = append(bindings, b)
	}
	return bindings, covered
}

// requireSkillPayload is the roster's separate payload check: an embedded Weapon
// must ship its skills/<id>/SKILL.md.
func requireSkillPayload(root string, b weaponBinding) weaponBinding {
	path := filepath.Join(root, "skills", b.SkillID, "SKILL.md")
	if info, err := os.Stat(path); err != nil || info.Size() == 0 {
		b.OK = false
		b.Reason = fmt.Sprintf("embedded weapon payload missing or empty: %s", path)
	}
	return b
}

// scanUncatalogedViews is the transitional scan: a skills/<id>/skill.yaml view for a
// Weapon the catalog does not list is still verified.
func scanUncatalogedViews(root, skillsDir string, entries []os.DirEntry, cataloged map[string]bool, roleSlotMap domain.RoleSlotMap, roleSlotMapErr error) []weaponBinding {
	var bindings []weaponBinding
	for _, entry := range entries {
		if cataloged[entry.Name()] {
			continue
		}
		if binding, ok := scanWeaponEntry(root, skillsDir, entry, roleSlotMap, roleSlotMapErr); ok {
			bindings = append(bindings, binding)
		}
	}
	return bindings
}

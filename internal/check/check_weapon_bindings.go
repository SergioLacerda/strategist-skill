package check

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// weaponBinding is the verification result for one embedded skill (weapon)
// that declares a canonical_role — DEC-003 of mission
// 20260913-wizard-hardcoded-fallback-maps-cleanup, recorded in
// docs/adr/0035-embedded-weapon-fallback-policy.md. Unlike
// resolveNativeFallback (which only discovers a compatible native-role
// fallback lazily, for whichever slot happens to already be configured as a
// skill_provider), this check runs unconditionally over every skill under
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
	CanonicalRole          string `yaml:"canonical_role"`
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
	skillsDir := filepath.Join(root, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("weapon bindings: read %s: %w", skillsDir, err)
	}

	// A missing/invalid roles/default.yaml is surfaced per-binding below
	// (each binding needing it fails with an explicit reason), not as a
	// function-level error — mirrors resolveNativeFallback's own treatment
	// of the same file.
	roleSlotMap, roleSlotMapErr := loadRoleSlotMap(root)

	var bindings []weaponBinding
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillID := entry.Name()
		manifestPath := filepath.Join(skillsDir, skillID, "skill.yaml")
		raw, readErr := os.ReadFile(manifestPath) //nolint:gosec // G304: path derived from the runtime skills directory
		if readErr != nil {
			continue // not a skill.yaml-bearing entry; nothing to verify
		}
		var taxonomy skillTaxonomy
		if yamlErr := yaml.Unmarshal(raw, &taxonomy); yamlErr != nil {
			bindings = append(bindings, weaponBinding{SkillID: skillID, Reason: fmt.Sprintf("skill.yaml invalid: %v", yamlErr)})
			continue
		}
		canonicalRole := taxonomy.canonicalRole()
		if canonicalRole == "" {
			continue // not claiming to be an embedded weapon for any role
		}
		bindings = append(bindings, verifyOneWeaponBinding(root, skillID, canonicalRole, roleSlotMap, roleSlotMapErr))
	}
	// The scan above is reactive (skill declares a role -> is the claim
	// valid?). It never asks the converse question a role-focused reading of
	// DEC-003 requires: does each permanent embedded-weapon pairing actually
	// exist at all? A pairing whose skill directory is entirely absent
	// produces zero rows from the scan above — silence, not a reported
	// failure. verifyEmbeddedWeaponRoster closes that direction.
	bindings = append(bindings, verifyEmbeddedWeaponRoster(bindings)...)
	return bindings, nil
}

// embeddedWeaponRoster is the permanent embedded weapon<->role pairing
// baseline fixed by docs/adr/0035-embedded-weapon-fallback-policy.md DEC-001:
// brainstorming<->ranger (discovery) and openspec-propose<->archivist
// (refinement). The execution slot's embedded weapon (paired with sniper) is
// explicitly deferred by that ADR and is intentionally not listed here.
var embeddedWeaponRoster = []struct {
	SkillID       string
	CanonicalRole string
}{
	{SkillID: "brainstorming", CanonicalRole: "ranger"},
	{SkillID: "openspec-propose", CanonicalRole: "archivist"},
}

// verifyEmbeddedWeaponRoster reports one failing weaponBinding for every
// embeddedWeaponRoster pairing whose SkillID does not appear at all among
// bindings already computed by the skill->role scan — i.e. its skill
// directory/manifest is missing from <root>/skills/ entirely. A pairing that
// is present but itself failing some other check (bad YAML, missing role
// file, slot mismatch) is already reported by that check and is not
// duplicated here: presence, not validity, is this function's only concern.
func verifyEmbeddedWeaponRoster(bindings []weaponBinding) []weaponBinding {
	present := make(map[string]bool, len(bindings))
	for _, b := range bindings {
		present[b.SkillID] = true
	}
	var missing []weaponBinding
	for _, expected := range embeddedWeaponRoster {
		if present[expected.SkillID] {
			continue
		}
		missing = append(missing, weaponBinding{
			SkillID:       expected.SkillID,
			CanonicalRole: expected.CanonicalRole,
			OK:            false,
			Reason: fmt.Sprintf(
				"embedded weapon missing from <root>/skills/ (expected permanent pairing %s<->%s, docs/adr/0035-embedded-weapon-fallback-policy.md DEC-001)",
				expected.SkillID, expected.CanonicalRole),
		})
	}
	return missing
}

func verifyOneWeaponBinding(root, skillID, canonicalRole string, roleSlotMap domain.RoleSlotMap, roleSlotMapErr error) weaponBinding {
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

	b.OK = true
	return b
}

// weaponBindingErrors renders every failed binding as a check.go-style error
// string, for inclusion in the same errs slice every other check.go
// validation gates the command's exit code on.
func weaponBindingErrors(bindings []weaponBinding) []string {
	var errs []string
	for _, b := range bindings {
		if !b.OK {
			errs = append(errs, fmt.Sprintf("weapon binding %s: %s", b.SkillID, b.Reason))
		}
	}
	return errs
}

package check

import "fmt"

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
// bindings already computed by the skill->role scan.
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

package check

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/catalog"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// rosterPairing is one Weapon<->Role pairing the compiled registry certifies
// for a skill-payload Weapon.
type rosterPairing struct {
	SkillID       string
	CanonicalRole string
}

// registryRosterID names the failing row reported when the compiled registry
// cannot be read, so an unreadable registry is never a silent empty roster.
const registryRosterID = "compiled-registry"

// registryRoster derives the expected embedded weapon<->role pairings from the
// compiled registry in <root>/plugins/catalog.yaml (ADR-0060 Decision 5). Ranked
// bindings whose Weapon runs as native role code (execution_mode code) are not
// skill payloads and are not scanned as Weapons.
func registryRoster(root string) ([]rosterPairing, error) {
	raw, err := os.ReadFile(filepath.Join(root, "plugins", "catalog.yaml")) //nolint:gosec // G304: fixed path under the runtime root
	if err != nil {
		return nil, fmt.Errorf("read compiled registry catalog: %w", err)
	}
	registry, err := catalog.ParseCompiledRegistryCatalog(raw)
	if err != nil {
		return nil, fmt.Errorf("parse compiled registry: %w", err)
	}
	pairings := make([]rosterPairing, 0, len(registry.RankedBindings))
	for _, binding := range registry.RankedBindings {
		if binding.Runtime.ExecutionMode == domain.WeaponExecutionModeCode {
			continue
		}
		pairings = append(pairings, rosterPairing{SkillID: binding.WeaponID, CanonicalRole: binding.Role})
	}
	return pairings, nil
}

// verifyEmbeddedWeaponRoster reports one failing weaponBinding for every
// registry-derived pairing whose SkillID does not appear at all among bindings
// already computed by the skill->role scan, or one failing row when the
// registry itself is unavailable.
func verifyEmbeddedWeaponRoster(root string, bindings []weaponBinding) []weaponBinding {
	expected, err := registryRoster(root)
	if err != nil {
		return []weaponBinding{{
			SkillID: registryRosterID,
			Reason:  fmt.Sprintf("compiled registry unavailable, permanent pairings cannot be derived (ADR-0060): %v", err),
		}}
	}
	present := make(map[string]bool, len(bindings))
	for _, b := range bindings {
		present[b.SkillID] = true
	}
	var missing []weaponBinding
	for _, pairing := range expected {
		if present[pairing.SkillID] {
			continue
		}
		missing = append(missing, weaponBinding{
			SkillID:       pairing.SkillID,
			CanonicalRole: pairing.CanonicalRole,
			OK:            false,
			Reason: fmt.Sprintf(
				"embedded weapon missing from <root>/skills/ (registry pairing %s<->%s, ADR-0060)",
				pairing.SkillID, pairing.CanonicalRole),
		})
	}
	return missing
}

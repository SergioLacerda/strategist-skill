// Package roster owns read-only ROSTER artifact construction. It deliberately
// depends only on domain contracts; installation and filesystem activation stay
// in their adapters.
package roster

import (
	"fmt"

	domainroster "github.com/SergioLacerda/strategist-skill/internal/domain/roster"
)

// Plan validates and materializes the deterministic ROSTER and selection
// projections. It performs no discovery, filesystem I/O, or activation.
func Plan(entries []domainroster.WeaponRosterEntry, selections []domainroster.WeaponSelectionArtifact) (domainroster.WeaponRosterArtifact, []domainroster.WeaponSelectionArtifact, error) {
	roster, err := domainroster.NewWeaponRosterArtifact(entries)
	if err != nil {
		return domainroster.WeaponRosterArtifact{}, nil, fmt.Errorf("roster: build artifact: %w", err)
	}
	for _, selection := range selections {
		if err := selection.Validate(); err != nil {
			return domainroster.WeaponRosterArtifact{}, nil, fmt.Errorf("roster: validate selection: %w", err)
		}
	}
	return roster, append([]domainroster.WeaponSelectionArtifact(nil), selections...), nil
}

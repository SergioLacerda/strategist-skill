// Package roster owns read-only ROSTER artifact construction. It deliberately
// depends only on domain contracts; installation and filesystem activation stay
// in their adapters.
package roster

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// Plan validates and materializes the deterministic ROSTER and selection
// projections. It performs no discovery, filesystem I/O, or activation.
func Plan(entries []domain.WeaponRosterEntry, selections []domain.WeaponSelectionArtifact) (domain.WeaponRosterArtifact, []domain.WeaponSelectionArtifact, error) {
	roster, err := domain.NewWeaponRosterArtifact(entries)
	if err != nil {
		return domain.WeaponRosterArtifact{}, nil, fmt.Errorf("roster: build artifact: %w", err)
	}
	for _, selection := range selections {
		if err := selection.Validate(); err != nil {
			return domain.WeaponRosterArtifact{}, nil, fmt.Errorf("roster: validate selection: %w", err)
		}
	}
	return roster, append([]domain.WeaponSelectionArtifact(nil), selections...), nil
}

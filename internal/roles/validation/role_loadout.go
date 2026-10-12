package rolevalidation

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/mechanisms"
)

// BuildRoleLoadout resolves the pinned Role→Weapon plan and composes the
// role-scoped Feats/Tools registry projection for the requested Stage.
func BuildRoleLoadout(root, slot string, resolution domain.StageResolution) (domain.RoleLoadout, error) {
	plan, err := BuildRoleInvocationPlan(root, slot)
	if err != nil {
		return domain.RoleLoadout{}, fmt.Errorf("role loadout: %w", err)
	}
	registry, err := mechanisms.Load(root)
	if err != nil {
		return domain.RoleLoadout{}, fmt.Errorf("role loadout: mechanisms registry: %w", err)
	}
	loadout, err := registry.BuildRoleLoadout(resolution, plan)
	if err != nil {
		return domain.RoleLoadout{}, fmt.Errorf("role loadout: %w", err)
	}
	return loadout, nil
}

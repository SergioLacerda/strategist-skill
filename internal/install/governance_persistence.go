package install

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/governance"
)

func persistGovernanceState(strategistDir string, policy *domain.TrustPolicy, grants *domain.PermissionGrantFile) error {
	if policy != nil {
		if err := governance.SavePolicy(strategistDir, *policy); err != nil {
			return fmt.Errorf("save trust policy: %w", err)
		}
	}
	return persistPermissionGrants(strategistDir, grants)
}

func persistPermissionGrants(strategistDir string, grants *domain.PermissionGrantFile) error {
	if grants == nil {
		return nil
	}
	if err := governance.SaveGrants(strategistDir, *grants); err != nil {
		return fmt.Errorf("save permission grants: %w", err)
	}
	return nil
}

package connectors

import (
	"context"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// ResolveRoleWeaponConnector selects the connector authorized by a resolved
// binding. Ranked Embedded bindings can only use the Strategist dispatch table;
// Custom bindings must receive an explicit external connector from the host.
func ResolveRoleWeaponConnector(ctx context.Context, binding domain.RoleWeaponBinding, embedded EmbeddedWeaponDispatch, custom RuntimeConnector) (RuntimeConnector, error) {
	switch binding.Mode {
	case domain.SlotBindingModeRanked:
		return resolveRankedConnector(ctx, binding, embedded)
	case domain.SlotBindingModeCustom:
		return resolveCustomConnector(ctx, binding, custom)
	default:
		return nil, fmt.Errorf("role/weapon connector: unsupported binding mode %q", binding.Mode)
	}
}

func resolveRankedConnector(ctx context.Context, binding domain.RoleWeaponBinding, embedded EmbeddedWeaponDispatch) (RuntimeConnector, error) {
	if binding.RuntimeKind != domain.RankedRuntimeEmbedded {
		return nil, fmt.Errorf("ranked_runtime_connector_unavailable: Ranked binding %s/%s declares runtime %q", binding.Role, binding.Slot, binding.RuntimeKind)
	}
	connector, err := embedded.Connector(binding.WeaponID, binding.WeaponVersion)
	if err != nil {
		return nil, err
	}
	if !connector.Capabilities(ctx).CanInvoke {
		return nil, fmt.Errorf("ranked_runtime_connector_unavailable: Embedded Weapon %q cannot invoke", domain.WeaponIdentity(binding.WeaponID, binding.WeaponVersion))
	}
	return connector, nil
}

func resolveCustomConnector(ctx context.Context, binding domain.RoleWeaponBinding, custom RuntimeConnector) (RuntimeConnector, error) {
	if custom == nil {
		return nil, fmt.Errorf("custom_runtime_connector_unavailable: no connector for Weapon %q", binding.WeaponID)
	}
	if !custom.Capabilities(ctx).CanInvoke {
		return nil, fmt.Errorf("custom_runtime_connector_unavailable: connector for Weapon %q cannot invoke", binding.WeaponID)
	}
	return custom, nil
}

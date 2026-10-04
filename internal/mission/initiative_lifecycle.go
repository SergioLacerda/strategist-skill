package mission

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/feats/initiative"
)

// NewDefaultRoleLifecycle creates the production role-boundary adapter using
// the canonical policy and the registry's declared INITIATIVE hooks.
func NewDefaultRoleLifecycle(strategistRoot string, registry domain.RoleRegistry) (RoleLifecycle, error) {
	runtime, err := NewDefaultInitiativeRuntime(strategistRoot)
	if err != nil {
		return RoleLifecycle{}, fmt.Errorf("initiative lifecycle: create runtime: %w", err)
	}
	return RoleLifecycle{runtime: runtime, registry: registry}, nil
}

// RoleLifecycle is the production adapter between the Scout entry declaration
// and the consultative runtime. Roles without an entry hook are not part of
// the current INITIATIVE flow.
type RoleLifecycle struct {
	runtime  InitiativeRuntime
	registry domain.RoleRegistry
}

// Enter invokes the declared INITIATIVE start hook for a role.
func (l RoleLifecycle) Enter(input InitiativeRoleEntry) (initiative.Advice, error) {
	hooks, ok := l.registry.InitiativeHooksOf(input.Role)
	if !ok || hooks.OnStart == "" {
		return initiative.Advice{}, nil
	}
	if hooks.OnStart != "resolve_advice" {
		return initiative.Advice{}, fmt.Errorf("initiative lifecycle: unsupported start hook %q", hooks.OnStart)
	}
	if input.Leveling == nil {
		return initiative.Advice{}, fmt.Errorf("initiative lifecycle: LEVELING resolution is required before INITIATIVE")
	}
	if err := input.Leveling.ValidateForRole(input.Role); err != nil {
		return initiative.Advice{}, fmt.Errorf("initiative lifecycle: validate LEVELING resolution: %w", err)
	}
	return l.runtime.EnterRole(input)
}

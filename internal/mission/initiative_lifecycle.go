package mission

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/initiative"
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

// RoleLifecycle is the production adapter between role declarations and the
// consultative runtime. A missing hook is a deliberate legacy no-op.
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

// Reevaluate invokes INITIATIVE only after LEVELING has emitted a fresh
// resolution. The prior resolution remains in the old Advice record.
func (l RoleLifecycle) Reevaluate(input InitiativeRoleEntry, trigger initiative.Trigger) (initiative.Advice, error) {
	hooks, ok := l.registry.InitiativeHooksOf(input.Role)
	if !ok || hooks.OnStart == "" {
		return initiative.Advice{}, nil
	}
	if hooks.OnStart != "resolve_advice" {
		return initiative.Advice{}, fmt.Errorf("initiative lifecycle: unsupported start hook %q", hooks.OnStart)
	}
	if input.Leveling == nil {
		return initiative.Advice{}, fmt.Errorf("initiative lifecycle: LEVELING resolution is required before INITIATIVE re-evaluation")
	}
	if err := input.Leveling.ValidateForRole(input.Role); err != nil {
		return initiative.Advice{}, fmt.Errorf("initiative lifecycle: validate LEVELING resolution: %w", err)
	}
	return l.runtime.Reevaluate(input, trigger)
}

// Complete invokes the declared INITIATIVE result hook for a role.
func (l RoleLifecycle) Complete(advice initiative.Advice, result initiative.Result, toRole string) (InitiativeHandoff, error) {
	hooks, ok := l.registry.InitiativeHooksOf(advice.Role)
	if !ok || hooks.OnResult == "" {
		return InitiativeHandoff{}, nil
	}
	if hooks.OnResult != "emit_initiative_result" {
		return InitiativeHandoff{}, fmt.Errorf("initiative lifecycle: unsupported result hook %q", hooks.OnResult)
	}
	return l.runtime.CompleteRole(advice, result, toRole)
}

// Consume invokes the declared INITIATIVE consume hook for a handoff.
func (l RoleLifecycle) Consume(handoff InitiativeHandoff) error {
	hooks, ok := l.registry.InitiativeHooksOf(handoff.ToRole)
	if !ok || hooks.OnStart == "" {
		return nil
	}
	if hooks.OnStart != "consume_advice" {
		return fmt.Errorf("initiative lifecycle: unsupported consume hook %q", hooks.OnStart)
	}
	return l.runtime.ConsumeHandoff(handoff)
}

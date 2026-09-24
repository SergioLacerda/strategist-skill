package domain

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// gateStep is the pipeline step between refinement and execution. It is not a
// role: it has no definition file, and its phase is derived from the role that
// owns the execution slot.
const gateStep = "gate"

// Role is the registry entry of one identity-bearing Strategist role (Scout,
// Ranger, Archivist, Sniper). Inline sub-routines are not roles.
type Role struct {
	ID string
	// Origin identifies ownership of the role contract, independently from
	// whether a provider may fill it.
	Origin RoleOrigin
	// Extensibility identifies whether a provider may fill the role.
	Extensibility RoleExtensibility
	// Slot is the active.yaml slot the role fills; empty for a pre-pipeline role.
	Slot string
	// Phase is the role's position in the mission checkpoint; 0 is pre-pipeline.
	Phase int
	// Pluggable is retained as a derived compatibility field for existing
	// consumers. New code should use Extensibility.
	Pluggable bool
	// HandoffSchema is the schema this role hands downstream; empty for a
	// terminal role.
	HandoffSchema string
	// Leveling names the LEVELING policy role to use; empty means the role id.
	Leveling string
	// OnStart lists command templates run when the role's phase starts; {role}
	// and {mission_id} are substituted, `<your-model>`/`<your-effort>` are a
	// literal reminder for the agent to fill in. Empty means DefaultStartCommand.
	OnStart []string
	// Initiative contains consultative hooks carried alongside, but independent
	// from, the role's LEVELING facts.
	Initiative InitiativeHooks
}

// RoleRegistry is the single authority for role facts (slot, phase order,
// pluggability, handoff schema and leveling key). Every consumer reads role
// facts from it instead of keeping its own role list.
type RoleRegistry struct {
	roles []Role // ordered by Phase, then ID
}

// DefaultRoleRegistry returns the built-in registry. It must stay identical to
// the embedded roles/*.yaml definitions (a parity test guards drift); runtime
// callers that know the workspace use LoadRoleRegistry to honor customizations.
func DefaultRoleRegistry() RoleRegistry {
	reg, err := NewRoleRegistry([]Role{
		{ID: "scout", Origin: RoleOriginNative, Extensibility: RoleExtensibilityFixed, OnStart: []string{DefaultStartCommand}, Initiative: InitiativeHooks{OnStart: "resolve_advice", OnResult: "emit_initiative_result", Preserve: []string{"advice_id", "policy_version", "policy_digest", "alignment", "evidence_refs"}}},
		{ID: "ranger", Origin: RoleOriginNative, Extensibility: RoleExtensibilityPluggable, Slot: string(SlotDiscovery), Phase: 1, Pluggable: true, HandoffSchema: "schemas/handoff-ranger-to-archivist.schema.yaml", OnStart: []string{DefaultStartCommand}, Initiative: InitiativeHooks{OnStart: "resolve_advice", OnResult: "emit_initiative_result", Preserve: []string{"advice_id", "policy_version", "policy_digest", "alignment", "evidence_refs"}}},
		{ID: "archivist", Origin: RoleOriginNative, Extensibility: RoleExtensibilityPluggable, Slot: string(SlotRefinement), Phase: 2, Pluggable: true, HandoffSchema: "schemas/handoff-archivist-to-sniper.schema.yaml", OnStart: []string{DefaultStartCommand}, Initiative: InitiativeHooks{OnStart: "consume_advice", OnResult: "emit_initiative_result", Preserve: []string{"advice_id", "policy_version", "policy_digest", "deviations", "evidence_refs"}}},
		{ID: "sniper", Origin: RoleOriginNative, Extensibility: RoleExtensibilityFixed, Slot: string(SlotExecution), Phase: 4, OnStart: []string{DefaultStartCommand}, Initiative: InitiativeHooks{OnStart: "consume_advice", OnResult: "emit_initiative_result", Preserve: []string{"advice_id", "policy_version", "policy_digest", "deviations", "evidence_refs"}}},
	})
	if err != nil {
		panic("domain: invalid built-in role registry: " + err.Error())
	}
	return reg
}

// NewRoleRegistry validates and orders a set of roles.
func NewRoleRegistry(roles []Role) (RoleRegistry, error) {
	index := registryIndex{ids: map[string]bool{}, phases: map[int]string{}}
	out := make([]Role, 0, len(roles))
	for _, role := range roles {
		role.ID = normalizeRoleID(role.ID)
		normalizeRoleTaxonomy(&role)
		if err := index.add(role); err != nil {
			return RoleRegistry{}, err
		}
		out = append(out, role)
	}
	sortRoles(out)
	return RoleRegistry{roles: out}, nil
}

// registryIndex tracks the ids and phases already taken while a registry is built.
type registryIndex struct {
	ids    map[string]bool
	phases map[int]string
}

func (x *registryIndex) add(role Role) error {
	if err := validateRegistryRole(role); err != nil {
		return err
	}
	if x.ids[role.ID] {
		return fmt.Errorf("role registry: duplicate role %q", role.ID)
	}
	x.ids[role.ID] = true
	if role.Phase == 0 {
		return nil
	}
	if other, taken := x.phases[role.Phase]; taken {
		return fmt.Errorf("role registry: roles %q and %q share phase %d", other, role.ID, role.Phase)
	}
	x.phases[role.Phase] = role.ID
	return nil
}

func normalizeRoleID(id string) string { return strings.ToLower(strings.TrimSpace(id)) }

func normalizeRoleTaxonomy(role *Role) {
	if role.Origin == "" {
		role.Origin = RoleOriginNative
	}
	if role.Extensibility == "" {
		if role.Pluggable {
			role.Extensibility = RoleExtensibilityPluggable
		} else {
			role.Extensibility = RoleExtensibilityFixed
		}
	}
	role.Pluggable = role.Extensibility.IsPluggable()
}

func sortRoles(roles []Role) {
	sort.SliceStable(roles, func(i, j int) bool {
		if roles[i].Phase != roles[j].Phase {
			return roles[i].Phase < roles[j].Phase
		}
		return roles[i].ID < roles[j].ID
	})
}

func validateRegistryRole(role Role) error {
	if err := ValidateRoleReference(role.ID); err != nil {
		return fmt.Errorf("role registry: %w", err)
	}
	switch {
	case role.ID == "":
		return errors.New("role registry: role id is required")
	case role.Phase < 0:
		return fmt.Errorf("role registry: role %q has a negative phase", role.ID)
	case role.Slot != "" && !IsValidSlot(role.Slot):
		return fmt.Errorf("role registry: role %q slot %q is not one of %s", role.ID, role.Slot, requiredSlotList)
	case role.Origin.Validate() != nil:
		return fmt.Errorf("role registry: role %q: %w", role.ID, role.Origin.Validate())
	case role.Extensibility.Validate() != nil:
		return fmt.Errorf("role registry: role %q: %w", role.ID, role.Extensibility.Validate())
	case role.Extensibility.IsPluggable() && role.Slot == "":
		return fmt.Errorf("role registry: pluggable role %q requires a slot", role.ID)
	}
	return nil
}

// RoleFromConfig derives a registry Role from a roles/<id>.yaml definition.
func RoleFromConfig(cfg RoleConfig) Role {
	return Role{
		ID: cfg.Role, Origin: cfg.EffectiveOrigin(), Extensibility: cfg.EffectiveExtensibility(), Slot: cfg.Slot, Phase: cfg.Phase,
		Pluggable:     cfg.EffectiveExtensibility().IsPluggable(),
		HandoffSchema: cfg.HandoffSchema, Leveling: cfg.Leveling, OnStart: cfg.OnStart,
		Initiative: cfg.Initiative,
	}
}

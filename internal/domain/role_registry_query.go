package domain

import "strings"

// Roles returns a copy of the ordered role list.
func (r RoleRegistry) Roles() []Role { return append([]Role(nil), r.roles...) }

// IDs returns the role ids ordered by phase.
func (r RoleRegistry) IDs() []string {
	ids := make([]string, len(r.roles))
	for i, role := range r.roles {
		ids[i] = role.ID
	}
	return ids
}

// Get returns a role by id (case-insensitive).
func (r RoleRegistry) Get(id string) (Role, bool) {
	want := strings.ToLower(strings.TrimSpace(id))
	for _, role := range r.roles {
		if role.ID == want {
			return role, true
		}
	}
	return Role{}, false
}

// Has reports whether id is a registered role.
func (r RoleRegistry) Has(id string) bool {
	_, ok := r.Get(id)
	return ok
}

// RoleForSlot returns the role that fills a slot.
func (r RoleRegistry) RoleForSlot(slot string) (Role, bool) {
	for _, role := range r.roles {
		if role.Slot != "" && role.Slot == slot {
			return role, true
		}
	}
	return Role{}, false
}

// PolicyRole returns the LEVELING policy role to use for id: the role's
// `leveling` key, defaulting to the id itself (also for unregistered ids).
func (r RoleRegistry) PolicyRole(id string) string {
	if role, ok := r.Get(id); ok && strings.TrimSpace(role.Leveling) != "" {
		return strings.ToLower(strings.TrimSpace(role.Leveling))
	}
	return strings.ToLower(strings.TrimSpace(id))
}

// HandoffSchemaOf returns the schema a role hands downstream, if any.
func (r RoleRegistry) HandoffSchemaOf(id string) string {
	role, _ := r.Get(id)
	return role.HandoffSchema
}

// InitiativeHooksOf returns the consultative hooks declared by a role.
func (r RoleRegistry) InitiativeHooksOf(id string) (InitiativeHooks, bool) {
	role, ok := r.Get(id)
	if !ok {
		return InitiativeHooks{}, false
	}
	return InitiativeHooks{
		OnStart: role.Initiative.OnStart, OnResult: role.Initiative.OnResult,
		Preserve: append([]string(nil), role.Initiative.Preserve...),
	}, true
}

// PhaseOf returns the checkpoint position of a role or of the approval gate,
// which always sits immediately before the execution role.
func (r RoleRegistry) PhaseOf(id string) (int, bool) {
	if strings.EqualFold(strings.TrimSpace(id), gateStep) {
		if exec, ok := r.RoleForSlot(string(SlotExecution)); ok && exec.Phase > 1 {
			return exec.Phase - 1, true
		}
		return 0, false
	}
	role, ok := r.Get(id)
	return role.Phase, ok
}

// PhaseTotal is the last checkpoint position, derived from the registry.
func (r RoleRegistry) PhaseTotal() int {
	total := 0
	for _, role := range r.roles {
		if role.Phase > total {
			total = role.Phase
		}
	}
	return total
}

// DefaultStartCommand resolves and records the role's level when its phase
// starts, so the model x effort label is part of role invocation rather than
// something the agent must remember. `<your-model>`/`<your-effort>` are a
// literal reminder for the invoking agent to fill in, not a substituted
// placeholder: the CLI cannot infer which model/effort is calling it, and an
// invocation with those flags omitted silently returns blank fields (see
// `20260922-strategist-ux-language-leveling-drift`). Only `{role}` and
// `{mission_id}` are mechanically substituted by StartCommands below.
const DefaultStartCommand = "strategist leveling label --role {role} --mission {mission_id} --host-model <your-model> --host-effort <your-effort>"

// MechanismsBriefCommand prints the role-scoped Mechanisms brief, so an agent
// starts a phase knowing which tools it has and how to invoke them.
const MechanismsBriefCommand = "strategist mechanisms brief --role {role}"

// DefaultStartCommands is what a role runs at phase start when it declares none:
// the level label, then the Mechanisms brief.
func DefaultStartCommands() []string {
	return []string{DefaultStartCommand, MechanismsBriefCommand}
}

// StartCommands returns the commands a role runs when its phase starts, with
// {role} and {mission_id} substituted (`<your-model>`/`<your-effort>` are left
// as-is for the invoking agent to replace with the actual running model and
// effort). An unregistered role has none.
func (r RoleRegistry) StartCommands(id, missionID string) []string {
	role, ok := r.Get(id)
	if !ok {
		return nil
	}
	templates := role.OnStart
	if len(templates) == 0 {
		templates = DefaultStartCommands()
	}
	replacer := strings.NewReplacer("{role}", role.ID, "{mission_id}", missionID)
	out := make([]string, len(templates))
	for i, template := range templates {
		out[i] = replacer.Replace(template)
	}
	return out
}

// LevelingRoleIDs lists the roles whose model x effort can be chosen, in phase
// order. It follows the built-in registry.
func LevelingRoleIDs() []string { return DefaultRoleRegistry().IDs() }

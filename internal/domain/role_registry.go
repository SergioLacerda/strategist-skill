package domain

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// gateStep is the pipeline step between refinement and execution. It is not a
// role: it has no definition file, and its phase is derived from the role that
// owns the execution slot.
const gateStep = "gate"

// Role is the registry entry of one identity-bearing Strategist role (Scout,
// Ranger, Archivist, Sniper). Inline sub-routines are not roles.
type Role struct {
	ID string
	// Slot is the active.yaml slot the role fills; empty for a pre-pipeline role.
	Slot string
	// Phase is the role's position in the mission checkpoint; 0 is pre-pipeline.
	Phase int
	// Pluggable records whether an external provider may fill the role today. It
	// changes nothing by itself.
	Pluggable bool
	// HandoffSchema is the schema this role hands downstream; empty for a
	// terminal role.
	HandoffSchema string
	// Leveling names the LEVELING policy role to use; empty means the role id.
	Leveling string
	// OnStart lists command templates run when the role's phase starts; {role}
	// and {mission_id} are substituted. Empty means DefaultStartCommand.
	OnStart []string
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
		{ID: "scout", OnStart: []string{DefaultStartCommand}},
		{ID: "ranger", Slot: string(SlotDiscovery), Phase: 1, Pluggable: true, HandoffSchema: "schemas/handoff-ranger-to-archivist.schema.yaml", OnStart: []string{DefaultStartCommand}},
		{ID: "archivist", Slot: string(SlotRefinement), Phase: 2, Pluggable: true, HandoffSchema: "schemas/handoff-archivist-to-sniper.schema.yaml", OnStart: []string{DefaultStartCommand}},
		{ID: "sniper", Slot: string(SlotExecution), Phase: 4, OnStart: []string{DefaultStartCommand}},
	})
	if err != nil {
		panic("domain: invalid built-in role registry: " + err.Error())
	}
	return reg
}

// NewRoleRegistry validates and orders a set of roles.
func NewRoleRegistry(roles []Role) (RoleRegistry, error) {
	seenID := map[string]bool{}
	seenPhase := map[int]string{}
	out := make([]Role, 0, len(roles))
	for _, role := range roles {
		role.ID = strings.ToLower(strings.TrimSpace(role.ID))
		if err := validateRegistryRole(role); err != nil {
			return RoleRegistry{}, err
		}
		if seenID[role.ID] {
			return RoleRegistry{}, fmt.Errorf("role registry: duplicate role %q", role.ID)
		}
		seenID[role.ID] = true
		if role.Phase > 0 {
			if other, dup := seenPhase[role.Phase]; dup {
				return RoleRegistry{}, fmt.Errorf("role registry: roles %q and %q share phase %d", other, role.ID, role.Phase)
			}
			seenPhase[role.Phase] = role.ID
		}
		out = append(out, role)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Phase != out[j].Phase {
			return out[i].Phase < out[j].Phase
		}
		return out[i].ID < out[j].ID
	})
	return RoleRegistry{roles: out}, nil
}

func validateRegistryRole(role Role) error {
	switch {
	case role.ID == "":
		return errors.New("role registry: role id is required")
	case role.Phase < 0:
		return fmt.Errorf("role registry: role %q has a negative phase", role.ID)
	case role.Slot != "" && !IsValidSlot(role.Slot):
		return fmt.Errorf("role registry: role %q slot %q is not one of %s", role.ID, role.Slot, requiredSlotList)
	case role.Pluggable && role.Slot == "":
		return fmt.Errorf("role registry: pluggable role %q requires a slot", role.ID)
	}
	return nil
}

// RoleFromConfig derives a registry Role from a roles/<id>.yaml definition.
func RoleFromConfig(cfg RoleConfig) Role {
	return Role{
		ID: cfg.Role, Slot: cfg.Slot, Phase: cfg.Phase,
		Pluggable:     cfg.Pluggable != nil && *cfg.Pluggable,
		HandoffSchema: cfg.HandoffSchema, Leveling: cfg.Leveling, OnStart: cfg.OnStart,
	}
}

// LoadRoleRegistry overlays the roles/*.yaml definitions found in dir on the
// built-in registry: a file replaces the role with the same id, and a new file
// adds a role. A missing directory yields the built-ins; a malformed file is an
// error. default.yaml (the slot map) is not a role definition.
func LoadRoleRegistry(dir string) (RoleRegistry, error) {
	merged := map[string]Role{}
	for _, role := range DefaultRoleRegistry().roles {
		merged[role.ID] = role
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultRoleRegistry(), nil
		}
		return RoleRegistry{}, fmt.Errorf("role registry: read %s: %w", dir, err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".yaml" || name == "default.yaml" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, name)) //nolint:gosec // fixed directory below .strategist
		if err != nil {
			return RoleRegistry{}, fmt.Errorf("role registry: read %s: %w", name, err)
		}
		var cfg RoleConfig
		if err := yaml.Unmarshal(raw, &cfg); err != nil {
			return RoleRegistry{}, fmt.Errorf("role registry: parse %s: %w", name, err)
		}
		role := RoleFromConfig(cfg)
		role.ID = strings.ToLower(strings.TrimSpace(role.ID))
		if role.ID == "" {
			return RoleRegistry{}, fmt.Errorf("role registry: %s: role is required", name)
		}
		merged[role.ID] = role
	}
	roles := make([]Role, 0, len(merged))
	for _, role := range merged {
		roles = append(roles, role)
	}
	return NewRoleRegistry(roles)
}

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
// something the agent must remember.
const DefaultStartCommand = "strategist leveling label --role {role} --mission {mission_id}"

// StartCommands returns the commands a role runs when its phase starts, with
// {role} and {mission_id} substituted. An unregistered role has none.
func (r RoleRegistry) StartCommands(id, missionID string) []string {
	role, ok := r.Get(id)
	if !ok {
		return nil
	}
	templates := role.OnStart
	if len(templates) == 0 {
		templates = []string{DefaultStartCommand}
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

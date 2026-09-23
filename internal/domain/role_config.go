package domain

import (
	"fmt"
	"strings"
)

// RoleConfig is the structure of a native role definition file (roles/<name>.yaml),
// e.g. roles/sniper.yaml — a role that declares its own slot and behavior contract
// instead of being backed by a skills/<provider>/skill.yaml manifest.
type RoleConfig struct {
	Role        string   `yaml:"role"`
	Slot        string   `yaml:"slot"`
	Must        []string `yaml:"must"`
	MustNot     []string `yaml:"must_not"`
	CustomBrief string   `yaml:"custom_brief"`
	// Phase is the role's checkpoint position (0 is pre-pipeline).
	Phase int `yaml:"phase,omitempty"`
	// Pluggable records whether an external provider may fill the role; nil means
	// unspecified (legacy files), which still requires a slot.
	Pluggable *bool `yaml:"pluggable,omitempty"`
	// Leveling names the LEVELING policy role to use; empty means the role id.
	Leveling string `yaml:"leveling,omitempty"`
	// HandoffSchema is the schema the role hands downstream; empty for a terminal role.
	HandoffSchema string `yaml:"handoff_schema,omitempty"`
	// OnStart lists command templates the role runs when its phase starts;
	// {role} and {mission_id} are substituted, `<your-model>`/`<your-effort>`
	// are a literal reminder for the agent to fill in. Empty means the
	// built-in default.
	OnStart []string `yaml:"on_start,omitempty"`
	// Initiative declares the internal consultative role-boundary hooks. It is
	// additive metadata and never changes LEVELING resolution.
	Initiative InitiativeHooks `yaml:"initiative,omitempty"`
}

// InitiativeHooks describes the advisory INITIATIVE lifecycle declared by a
// role definition.
type InitiativeHooks struct {
	OnStart  string   `yaml:"on_start,omitempty"`
	OnResult string   `yaml:"on_result,omitempty"`
	Preserve []string `yaml:"preserve,omitempty"`
}

// Validate returns an error if the role definition is missing required fields
// or declares an unknown slot.
func (r RoleConfig) Validate() error {
	var errs []string
	if r.Role == "" {
		errs = append(errs, "role is required")
	}
	if r.Slot == "" {
		// A role without a slot cannot be plugged, so it must say so explicitly.
		if r.Pluggable == nil || *r.Pluggable {
			errs = append(errs, "slot is required")
		}
	} else if !IsValidSlot(r.Slot) {
		errs = append(errs, fmt.Sprintf("slot %q is not one of %s", r.Slot, requiredSlotList))
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("role config invalid: %s", strings.Join(errs, "; "))
}

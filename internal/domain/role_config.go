package domain

import (
	"fmt"
	"strings"
)

// RoleConfig is the structure of a native role definition file (roles/<name>.yaml),
// e.g. roles/sniper.yaml — a role that declares its own slot and behavior contract
// instead of being backed by a skills/<provider>/skill.yaml manifest.
type RoleConfig struct {
	Role          string            `yaml:"role"`
	Slot          string            `yaml:"slot"`
	Origin        RoleOrigin        `yaml:"origin,omitempty"`
	Extensibility RoleExtensibility `yaml:"extensibility,omitempty"`
	Must          []string          `yaml:"must"`
	MustNot       []string          `yaml:"must_not"`
	CustomBrief   string            `yaml:"custom_brief"`
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
	} else if err := ValidateRoleReference(r.Role); err != nil {
		errs = append(errs, err.Error())
	}
	errs = append(errs, r.taxonomyErrors()...)
	if msg := r.slotError(); msg != "" {
		errs = append(errs, msg)
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("role config invalid: %s", strings.Join(errs, "; "))
}

// taxonomyErrors reports invalid origin/extensibility values and a legacy
// pluggable flag that contradicts an explicit extensibility.
func (r RoleConfig) taxonomyErrors() []string {
	var errs []string
	if err := r.EffectiveOrigin().Validate(); err != nil {
		errs = append(errs, err.Error())
	}
	extensibility := r.EffectiveExtensibility()
	if err := extensibility.Validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if r.Extensibility != "" && r.Pluggable != nil && *r.Pluggable != extensibility.IsPluggable() {
		errs = append(errs, "pluggable conflicts with extensibility")
	}
	return errs
}

// slotError returns the slot violation, or "" when the slot declaration is valid.
func (r RoleConfig) slotError() string {
	if r.Slot != "" {
		if IsValidSlot(r.Slot) {
			return ""
		}
		return fmt.Sprintf("slot %q is not one of %s", r.Slot, requiredSlotList)
	}
	// Legacy files without either taxonomy field must still make the
	// slotless/fixed decision explicit.
	if r.Extensibility == "" && r.Pluggable == nil {
		return "slot is required or extensibility: fixed must be explicit"
	}
	if r.EffectiveExtensibility().IsPluggable() {
		return "slot is required"
	}
	return ""
}

// EffectiveOrigin returns the canonical origin while keeping legacy role files
// readable during the taxonomy migration.
func (r RoleConfig) EffectiveOrigin() RoleOrigin {
	if r.Origin == "" {
		return RoleOriginNative
	}
	return r.Origin
}

// EffectiveExtensibility returns the canonical extensibility. The old
// pluggable pointer remains a compatibility input only.
func (r RoleConfig) EffectiveExtensibility() RoleExtensibility {
	if r.Extensibility != "" {
		return r.Extensibility
	}
	if r.Pluggable != nil && *r.Pluggable {
		return RoleExtensibilityPluggable
	}
	return RoleExtensibilityFixed
}

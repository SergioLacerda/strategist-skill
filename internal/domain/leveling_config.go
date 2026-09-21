package domain

import (
	"fmt"
	"sort"
	"strings"
)

// LEVELING decision modes stored in active.yaml.
const (
	LevelingModeAutomatic = "automatic"
	LevelingModeManual    = "manual"
)

// LevelingEffortTiers are the effort tiers accepted for a manual choice. They
// mirror internal/leveling's tier catalog (a parity test guards drift).
var LevelingEffortTiers = []string{"none", "low", "medium", "high", "xhigh", "max"}

// LevelingRoleChoice is the operator's manual model and effort for one role.
type LevelingRoleChoice struct {
	Model  string `yaml:"model,omitempty"`
	Effort string `yaml:"effort,omitempty"`
}

// Complete reports whether both model and effort are set, so nothing needs to
// be read from the LEVELING policy.
func (c LevelingRoleChoice) Complete() bool {
	return strings.TrimSpace(c.Model) != "" && strings.TrimSpace(c.Effort) != ""
}

// LevelingConfig is the optional `leveling:` block of active.yaml. An absent
// block (zero value) behaves as automatic.
type LevelingConfig struct {
	Mode  string                        `yaml:"mode,omitempty"`
	Roles map[string]LevelingRoleChoice `yaml:"roles,omitempty"`
}

// EffectiveMode returns the normalized mode; empty means automatic.
func (c LevelingConfig) EffectiveMode() string {
	mode := strings.ToLower(strings.TrimSpace(c.Mode))
	if mode == "" {
		return LevelingModeAutomatic
	}
	return mode
}

// Choice returns the manual choice for a role. It reports false unless the mode
// is manual and the role has an entry.
func (c LevelingConfig) Choice(role string) (LevelingRoleChoice, bool) {
	if c.EffectiveMode() != LevelingModeManual {
		return LevelingRoleChoice{}, false
	}
	want := strings.ToLower(strings.TrimSpace(role))
	for id, choice := range c.Roles {
		if strings.ToLower(strings.TrimSpace(id)) == want {
			return choice, true
		}
	}
	return LevelingRoleChoice{}, false
}

// Validate checks the block; it is called from ActiveConfig.Validate.
func (c LevelingConfig) Validate() error {
	switch c.EffectiveMode() {
	case LevelingModeAutomatic:
		return nil
	case LevelingModeManual:
	default:
		return fmt.Errorf("leveling_mapping_invalid: leveling.mode %q must be %q or %q", c.Mode, LevelingModeAutomatic, LevelingModeManual)
	}
	if len(c.Roles) == 0 {
		return fmt.Errorf("leveling_mapping_invalid: leveling.roles requires at least one role when mode is %q", LevelingModeManual)
	}
	ids := make([]string, 0, len(c.Roles))
	for id := range c.Roles {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err := validateLevelingRole(id, c.Roles[id]); err != nil {
			return err
		}
	}
	return nil
}

func validateLevelingRole(id string, choice LevelingRoleChoice) error {
	if !DefaultRoleRegistry().Has(id) {
		return fmt.Errorf("leveling_mapping_invalid: leveling.roles.%s: unknown role (want one of %s)", id, strings.Join(LevelingRoleIDs(), ", "))
	}
	model, effort := strings.TrimSpace(choice.Model), strings.TrimSpace(choice.Effort)
	if model == "" && effort == "" {
		return fmt.Errorf("leveling_mapping_invalid: leveling.roles.%s requires a model or an effort", id)
	}
	if strings.ContainsAny(model, "\r\n") {
		return fmt.Errorf("leveling_mapping_invalid: leveling.roles.%s.model must be a single-line name", id)
	}
	if effort != "" && !containsString(LevelingEffortTiers, strings.ToLower(effort)) {
		return fmt.Errorf("leveling_mapping_invalid: leveling.roles.%s.effort %q is not one of %s", id, choice.Effort, strings.Join(LevelingEffortTiers, ", "))
	}
	return nil
}

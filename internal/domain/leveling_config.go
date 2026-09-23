package domain

import (
	"fmt"
	"strings"
)

// LEVELING decision modes stored in active.yaml.
const (
	LevelingModeAutomatic = "automatic"
	LevelingModeManual    = "manual"
)

// LevelingEffortTiers are the effort tiers accepted from a host report. They
// mirror internal/leveling's tier catalog (a parity test guards drift).
var LevelingEffortTiers = []string{"none", "low", "medium", "high", "xhigh", "max"}

// LevelingConfig is the optional `leveling:` block of active.yaml. An absent
// block (zero value) behaves as automatic. Manual stores no per-role values: the
// host's model and effort are used as-is.
type LevelingConfig struct {
	Mode string `yaml:"mode,omitempty"`
}

// EffectiveMode returns the normalized mode; empty means automatic.
func (c LevelingConfig) EffectiveMode() string {
	mode := strings.ToLower(strings.TrimSpace(c.Mode))
	if mode == "" {
		return LevelingModeAutomatic
	}
	return mode
}

// HostPassthrough reports whether role levels come only from the host, with
// the LEVELING policy never loaded or applied (manual mode).
func (c LevelingConfig) HostPassthrough() bool {
	return c.EffectiveMode() == LevelingModeManual
}

// Validate checks that the mode is one of the supported values.
func (c LevelingConfig) Validate() error {
	switch c.EffectiveMode() {
	case LevelingModeAutomatic, LevelingModeManual:
		return nil
	default:
		return fmt.Errorf("leveling_mapping_invalid: leveling.mode %q must be %q or %q", c.Mode, LevelingModeAutomatic, LevelingModeManual)
	}
}

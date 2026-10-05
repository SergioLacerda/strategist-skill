package domain

import (
	"fmt"
	"path/filepath"
	"strings"
)

// LEVELING decision modes stored in active.yaml.
const (
	LevelingModeAutomatic = "automatic"
	LevelingModeManual    = "manual"
	// DefaultBudgetLedgerPath keeps effort events separate from the role-level
	// memory ledger and other mission memory.
	DefaultBudgetLedgerPath = "memory-effort/budget-events.jsonl"
	// ReasonLevelingBudgetLedgerPathInvalid is stable because callers expose it
	// when active.yaml points outside the Strategist root.
	ReasonLevelingBudgetLedgerPathInvalid = "leveling_budget_ledger_path_invalid"
)

// LevelingEffortTiers are the effort tiers accepted from a host report. They
// mirror internal/tools/leveling's tier catalog (a parity test guards drift).
var LevelingEffortTiers = []string{"none", "low", "medium", "high", "xhigh", "max"}

// LevelingConfig is the optional `leveling:` block of active.yaml. An absent
// block (zero value) behaves as automatic. Manual stores no per-role values: the
// host's model and effort are used as-is.
type LevelingConfig struct {
	Mode             string `yaml:"mode,omitempty"`
	BudgetLedgerPath string `yaml:"budget_ledger_path,omitempty"`
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

// EffectiveBudgetLedgerPath returns the configured relative path or the
// isolated default path when no override is present.
func (c LevelingConfig) EffectiveBudgetLedgerPath() string {
	path := strings.TrimSpace(c.BudgetLedgerPath)
	if path == "" {
		return DefaultBudgetLedgerPath
	}
	return path
}

// Validate checks that the mode is one of the supported values.
func (c LevelingConfig) Validate() error {
	if err := ValidateBudgetLedgerPath(c.BudgetLedgerPath); err != nil {
		return err
	}
	switch c.EffectiveMode() {
	case LevelingModeAutomatic, LevelingModeManual:
		return nil
	default:
		return fmt.Errorf("leveling_mapping_invalid: leveling.mode %q must be %q or %q", c.Mode, LevelingModeAutomatic, LevelingModeManual)
	}
}

// ValidateBudgetLedgerPath permits only a project-relative path. Runtime
// callers join it with the Strategist root after this validation.
func ValidateBudgetLedgerPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%s: leveling.budget_ledger_path must stay below the Strategist root", ReasonLevelingBudgetLedgerPathInvalid)
	}
	return nil
}

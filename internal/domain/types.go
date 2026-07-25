package domain

import (
	"fmt"
	"strings"
)

// PhaseLabels holds the display name for each mission phase.
type PhaseLabels struct {
	Discovery  string `yaml:"discovery"`
	Refinement string `yaml:"refinement"`
	Execution  string `yaml:"execution"`
}

// ActiveConfig is the structure of a standalone active.yaml template.
type ActiveConfig struct {
	Mode               string `yaml:"mode"`
	BasePath           string `yaml:"base_path"`
	KnowledgeIndexPath string `yaml:"knowledge_index_path"`
	Language           any    `yaml:"language,omitempty"`
	// Legacy fields — parsed to detect stale active.yaml files. ValidateNoLegacyFields returns
	// an error if either is set, directing users to remove them.
	ExecutionMode      string            `yaml:"execution_mode,omitempty"`
	GitPersistenceMode string            `yaml:"git_persistence_mode,omitempty"`
	Slots              map[string]string `yaml:"slots"`
}

// ValidateNoLegacyFields returns an error if the config contains removed fields.
func (c ActiveConfig) ValidateNoLegacyFields() error {
	if c.ExecutionMode != "" {
		return fmt.Errorf("legacy field execution_mode is no longer supported; remove it from active.yaml")
	}
	if c.GitPersistenceMode != "" {
		return fmt.Errorf("legacy field git_persistence_mode is no longer supported; remove it from active.yaml")
	}
	return nil
}

// PersonaDiagnostics holds the bootstrap banner templates from a persona file.
type PersonaDiagnostics struct {
	Format          string `yaml:"format"`
	PipelineHeader  string `yaml:"pipeline_header"`
	BootstrapOrigin string `yaml:"bootstrap_origin"`
}

// PersonaConfig is the structure of a persona yaml file (personas/*.yaml).
type PersonaConfig struct {
	ID             string             `yaml:"id"`
	Description    string             `yaml:"description"`
	PhaseLabels    PhaseLabels        `yaml:"phase_labels"`
	ToneDirective  string             `yaml:"tone_directive"`
	ProgressPrefix string             `yaml:"progress_prefix"`
	Diagnostics    PersonaDiagnostics `yaml:"diagnostics"`
}

// ValidateForRuntime checks all fields required for CLI bootstrap and check validation.
func (p PersonaConfig) ValidateForRuntime() error {
	var errs []string
	if p.ID == "" {
		errs = append(errs, "id is required")
	}
	if p.ToneDirective == "" {
		errs = append(errs, "tone_directive is required")
	}
	if p.PhaseLabels.Discovery == "" || p.PhaseLabels.Refinement == "" || p.PhaseLabels.Execution == "" {
		errs = append(errs, "phase_labels.discovery/refinement/execution are required")
	}
	if p.Diagnostics.PipelineHeader == "" {
		errs = append(errs, "diagnostics.pipeline_header is required")
	}
	if p.Diagnostics.BootstrapOrigin == "" {
		errs = append(errs, "diagnostics.bootstrap_origin is required")
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("persona config invalid: %s", strings.Join(errs, "; "))
}

// validSlots are the three pipeline slots a role or role-slot-map entry may declare.
var validSlots = map[string]bool{"discovery": true, "refinement": true, "execution": true}

// RoleConfig is the structure of a native role definition file (roles/<name>.yaml),
// e.g. roles/sniper.yaml — a role that declares its own slot and behavior contract
// instead of being backed by a skills/<provider>/skill.yaml manifest.
type RoleConfig struct {
	Role        string   `yaml:"role"`
	Slot        string   `yaml:"slot"`
	Must        []string `yaml:"must"`
	MustNot     []string `yaml:"must_not"`
	CustomBrief string   `yaml:"custom_brief"`
}

// Validate returns an error if the role definition is missing required fields
// or declares an unknown slot.
func (r RoleConfig) Validate() error {
	var errs []string
	if r.Role == "" {
		errs = append(errs, "role is required")
	}
	if r.Slot == "" {
		errs = append(errs, "slot is required")
	} else if !validSlots[r.Slot] {
		errs = append(errs, fmt.Sprintf("slot %q is not one of discovery, refinement, execution", r.Slot))
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("role config invalid: %s", strings.Join(errs, "; "))
}

// RoleSlotMap is the structure of roles/default.yaml — a slot→provider mapping,
// mirroring the shape of active.yaml's slots field.
type RoleSlotMap map[string]string

// Validate returns an error if any of the three required slots is missing or empty.
func (m RoleSlotMap) Validate() error {
	var errs []string
	for _, slot := range []string{"discovery", "refinement", "execution"} {
		if m[slot] == "" {
			errs = append(errs, fmt.Sprintf("missing slot: %s", slot))
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("role slot map invalid: %s", strings.Join(errs, "; "))
}

// ApprovalGateContract is the structure of contracts/approval-gate.yaml.
type ApprovalGateContract struct {
	Module      string `yaml:"module"`
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
}

// Validate returns an error if any required field is missing or invalid.
// Called after unmarshal in the bootstrap fast path to catch config drift early.
func (c ActiveConfig) Validate() error {
	var errs []string
	if c.Mode == "" {
		errs = append(errs, "mode is required")
	}
	if c.BasePath == "" {
		errs = append(errs, "base_path is required")
	}
	if len(c.Slots) == 0 {
		errs = append(errs, "slots must have at least one entry")
	}
	// Execution policy is fixed — no per-config validation needed.
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("active config invalid: %s", strings.Join(errs, "; "))
}

// Validate returns an error if any required field is missing.
func (p PersonaConfig) Validate() error {
	var errs []string
	if p.ID == "" {
		errs = append(errs, "id is required")
	}
	if p.ToneDirective == "" {
		errs = append(errs, "tone_directive is required")
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("persona config invalid: %s", strings.Join(errs, "; "))
}

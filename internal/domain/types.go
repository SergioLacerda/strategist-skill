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
	Mode               string   `yaml:"mode"`
	BasePath           string   `yaml:"base_path"`
	DocumentationRoots []string `yaml:"documentation_roots,omitempty"`
	KnowledgeIndexPath string   `yaml:"knowledge_index_path"`
	Language           any      `yaml:"language,omitempty"`
	// Legacy fields — parsed to detect stale active.yaml files. ValidateNoLegacyFields returns
	// an error if either is set, directing users to remove them.
	ExecutionMode      string            `yaml:"execution_mode,omitempty"`
	GitPersistenceMode string            `yaml:"git_persistence_mode,omitempty"`
	Slots              map[string]string `yaml:"slots"`
	// ProviderResolutionPolicy is retained only to detect stale installations.
	// Active routing never applies it; non-empty values are rejected below.
	ProviderResolutionPolicy ResolutionPolicy `yaml:"provider_resolution_policy,omitempty"`
	// Leveling is the optional operator choice between manual and automatic
	// model x effort per role. Absent means automatic.
	Leveling LevelingConfig `yaml:"leveling,omitempty"`
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
	errs := append(p.requiredFieldErrors(), p.Diagnostics.runtimeErrors()...)
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("persona config invalid: %s", strings.Join(errs, "; "))
}

func (p PersonaConfig) requiredFieldErrors() []string {
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
	return errs
}

// runtimeErrors checks the bootstrap banner templates. format: jsonl personas
// (e.g. debug) bypass all profile/narrative rendering by design — every event
// is emitted as a structured JSON line instead, so a
// pipeline_header/bootstrap_origin banner template is never read and is not
// required.
func (d PersonaDiagnostics) runtimeErrors() []string {
	if d.Format == "jsonl" {
		return nil
	}
	var errs []string
	if d.PipelineHeader == "" {
		errs = append(errs, "diagnostics.pipeline_header is required")
	}
	if d.BootstrapOrigin == "" {
		errs = append(errs, "diagnostics.bootstrap_origin is required")
	}
	return errs
}

// RoleSlotMap is the structure of roles/default.yaml — a slot→provider mapping,
// mirroring the shape of active.yaml's slots field. A skill provider resolves at
// skills/<provider>/skill.yaml; native roles resolve through roles/<provider>.yaml.
type RoleSlotMap map[string]string

// Validate returns an error if any of the three required slots is missing or empty.
func (m RoleSlotMap) Validate() error {
	var errs []string
	for _, required := range RequiredSlots() {
		slot := string(required)
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
	errs = append(errs, validateActiveConfigSlots(c.Slots)...)
	if err := c.ProviderResolutionPolicy.Validate(); err != nil {
		errs = append(errs, err.Error())
	}
	if c.ProviderResolutionPolicy != "" {
		errs = append(errs, "provider_resolution_policy is retired; remove it from active.yaml")
	}
	if err := c.Leveling.Validate(); err != nil {
		errs = append(errs, err.Error())
	}
	// Execution policy is fixed — no per-config validation needed.
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("active config invalid: %s", strings.Join(errs, "; "))
}

func validateActiveConfigSlots(slots RoleSlotMap) []string {
	if len(slots) == 0 {
		return []string{"slots must have at least one entry"}
	}
	var errs []string
	for _, required := range RequiredSlots() {
		slot := string(required)
		if slots[slot] == "" {
			errs = append(errs, fmt.Sprintf("missing slot: %s", slot))
		}
	}
	return errs
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

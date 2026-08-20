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
	// ProviderResolutionPolicy governs what happens when a configured skill_provider
	// slot passes static `strategist check` validation but a compatible native role
	// also exists — see ResolutionPolicy and docs/adr/0028-native-role-resilient-baseline.md.
	// Omitted or empty is valid and means EffectivePolicy() applies DefaultResolutionPolicy.
	ProviderResolutionPolicy ResolutionPolicy `yaml:"provider_resolution_policy,omitempty"`
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
	} else if !IsValidSlot(r.Slot) {
		errs = append(errs, fmt.Sprintf("slot %q is not one of %s", r.Slot, requiredSlotList))
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
	// Execution policy is fixed — no per-config validation needed.
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("active config invalid: %s", strings.Join(errs, "; "))
}

// ResolutionPolicy controls agent behavior when a configured slot provider passes
// static `strategist check` validation (valid skill.yaml, matching risk_score) but
// turns out not to be invocable at mission time, and a compatible native role
// exists for the same slot. See docs/adr/0028-native-role-resilient-baseline.md.
type ResolutionPolicy string

const (
	// ResolutionPolicyBlock preserves strict failure behavior: role_invocation_failed
	// stops the mission, exactly as before ADR-0028. No native fallback is offered.
	ResolutionPolicyBlock ResolutionPolicy = "block"
	// ResolutionPolicyAsk requests explicit user confirmation before using the
	// compatible native role for this mission. This is the recommended default.
	ResolutionPolicyAsk ResolutionPolicy = "ask"
	// ResolutionPolicyNative uses the compatible native role automatically, while
	// requiring the agent to emit degradation evidence (configured provider,
	// effective provider, reason). Never implies Approval Gate acceptance.
	ResolutionPolicyNative ResolutionPolicy = "native"
)

// DefaultResolutionPolicy is applied when active.yaml omits provider_resolution_policy
// or sets it to the empty string. ADR-0028 recommends "ask" as the default.
const DefaultResolutionPolicy = ResolutionPolicyAsk

var validResolutionPolicies = map[ResolutionPolicy]bool{
	ResolutionPolicyBlock:  true,
	ResolutionPolicyAsk:    true,
	ResolutionPolicyNative: true,
}

// Validate returns an error if the policy is set to an unrecognized value. An
// empty policy is valid — EffectivePolicy resolves it to DefaultResolutionPolicy.
func (p ResolutionPolicy) Validate() error {
	if p == "" {
		return nil
	}
	if !validResolutionPolicies[p] {
		return fmt.Errorf("provider_resolution_policy %q is not one of block, ask, native", p)
	}
	return nil
}

// EffectivePolicy returns p, or DefaultResolutionPolicy when p is empty.
func (p ResolutionPolicy) EffectivePolicy() ResolutionPolicy {
	if p == "" {
		return DefaultResolutionPolicy
	}
	return p
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

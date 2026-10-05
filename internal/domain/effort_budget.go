package domain

import (
	"fmt"
	"strings"
)

// EffortBudgetSchemaVersion identifies the additive LEVELING orchestration
// envelope. It is deliberately separate from the model/effort artifact
// version so legacy consumers can continue reading their projection.
const EffortBudgetSchemaVersion = "strategist-effort-budget/v1"

// BudgetPermissions are the bounded kinds of work a Role may request.
type BudgetPermissions struct {
	Specialists bool `json:"specialists" yaml:"specialists"`
	SideQuests  bool `json:"side_quests" yaml:"side_quests"`
	DeepHistory bool `json:"deep_history" yaml:"deep_history"`
}

// EffortBudgetPolicy is the customer-editable, provider-neutral budget
// attached to a LEVELING role. It contains limits, not an execution plan.
type EffortBudgetPolicy struct {
	MaxDelegations    int               `json:"max_delegations" yaml:"max_delegations"`
	MaxDepth          int               `json:"max_depth" yaml:"max_depth"`
	Parallelism       int               `json:"parallelism" yaml:"parallelism"`
	ContextBudget     string            `json:"context_budget" yaml:"context_budget"`
	EvidenceDepth     string            `json:"evidence_depth" yaml:"evidence_depth"`
	Permissions       BudgetPermissions `json:"permissions" yaml:"permissions"`
	EscalationAllowed bool              `json:"escalation_allowed" yaml:"escalation_allowed"`
	MaxEscalations    int               `json:"max_escalations" yaml:"max_escalations"`
}

// EffortBudget is the immutable envelope issued by LEVELING and consumed by
// the runtime boundary. The identity fields make replay and escalation
// correlation explicit without granting INITIATIVE mutation authority.
type EffortBudget struct {
	SchemaVersion      string            `json:"schema_version" yaml:"schema_version"`
	MissionID          string            `json:"mission_id,omitempty" yaml:"mission_id,omitempty"`
	RunID              string            `json:"run_id,omitempty" yaml:"run_id,omitempty"`
	Role               string            `json:"role" yaml:"role"`
	Provider           string            `json:"provider,omitempty" yaml:"provider,omitempty"`
	Model              string            `json:"model,omitempty" yaml:"model,omitempty"`
	Capability         string            `json:"capability,omitempty" yaml:"capability,omitempty"`
	Source             string            `json:"source,omitempty" yaml:"source,omitempty"`
	MaxDelegations     int               `json:"max_delegations" yaml:"max_delegations"`
	MaxDepth           int               `json:"max_depth" yaml:"max_depth"`
	Parallelism        int               `json:"parallelism" yaml:"parallelism"`
	ContextBudget      string            `json:"context_budget" yaml:"context_budget"`
	EvidenceDepth      string            `json:"evidence_depth" yaml:"evidence_depth"`
	Permissions        BudgetPermissions `json:"permissions" yaml:"permissions"`
	EscalationAllowed  bool              `json:"escalation_allowed" yaml:"escalation_allowed"`
	MaxEscalations     int               `json:"max_escalations" yaml:"max_escalations"`
	EscalationLevel    int               `json:"escalation_level,omitempty" yaml:"escalation_level,omitempty"`
	ParentResolutionID string            `json:"parent_resolution_id,omitempty" yaml:"parent_resolution_id,omitempty"`
	ResolutionID       string            `json:"resolution_id,omitempty" yaml:"resolution_id,omitempty"`
	EscalationReason   string            `json:"escalation_reason,omitempty" yaml:"escalation_reason,omitempty"`
	PolicyVersion      int               `json:"policy_version,omitempty" yaml:"policy_version,omitempty"`
	PolicyDigest       string            `json:"policy_digest,omitempty" yaml:"policy_digest,omitempty"`
	FallbackUsed       bool              `json:"fallback_used,omitempty" yaml:"fallback_used,omitempty"`
	FallbackReason     string            `json:"fallback_reason,omitempty" yaml:"fallback_reason,omitempty"`
}

// NewEffortBudget projects a policy into a resolved, provenance-bearing
// envelope. It does not validate provider execution; the caller decides
// whether a provider mapping is authoritative before constructing it.
func NewEffortBudget(policy EffortBudgetPolicy, role, provider, model, capability, source string, version int, digest string) EffortBudget {
	return EffortBudget{
		SchemaVersion: EffortBudgetSchemaVersion, Role: strings.ToLower(strings.TrimSpace(role)),
		Provider: provider, Model: model, Capability: capability, Source: source,
		MaxDelegations: policy.MaxDelegations, MaxDepth: policy.MaxDepth, Parallelism: policy.Parallelism,
		ContextBudget: policy.ContextBudget, EvidenceDepth: policy.EvidenceDepth, Permissions: policy.Permissions,
		EscalationAllowed: policy.EscalationAllowed, MaxEscalations: policy.MaxEscalations,
		PolicyVersion: version, PolicyDigest: digest,
	}
}

// Validate checks the closed envelope and its provenance invariants.
func (b EffortBudget) Validate() error {
	if b.SchemaVersion != EffortBudgetSchemaVersion {
		return fmt.Errorf("effort budget: unsupported schema_version %q", b.SchemaVersion)
	}
	if err := validateEffortBudgetIdentity(b); err != nil {
		return err
	}
	if err := validateEffortBudgetLimits(b); err != nil {
		return err
	}
	if err := validateEffortBudgetPolicy(b); err != nil {
		return err
	}
	return validateEffortBudgetEscalation(b)
}

func validateEffortBudgetIdentity(b EffortBudget) error {
	if strings.TrimSpace(b.Role) == "" {
		return fmt.Errorf("effort budget: role is required")
	}
	return nil
}

func validateEffortBudgetLimits(b EffortBudget) error {
	if b.MaxDelegations < 0 || b.MaxDepth < 0 || b.Parallelism < 1 || b.MaxEscalations < 0 {
		return fmt.Errorf("effort budget: limits must be non-negative and parallelism must be positive")
	}
	return nil
}

func validateEffortBudgetPolicy(b EffortBudget) error {
	if !containsBudgetValue([]string{"low", "medium", "high"}, b.ContextBudget) {
		return fmt.Errorf("effort budget: unsupported context_budget %q", b.ContextBudget)
	}
	if !containsBudgetValue([]string{"minimal", "standard", "deep"}, b.EvidenceDepth) {
		return fmt.Errorf("effort budget: unsupported evidence_depth %q", b.EvidenceDepth)
	}
	if b.PolicyVersion < 0 {
		return fmt.Errorf("effort budget: policy_version must not be negative")
	}
	if b.PolicyVersion > 0 && strings.TrimSpace(b.PolicyDigest) == "" {
		return fmt.Errorf("effort budget: policy_digest is required with policy_version")
	}
	return nil
}

func validateEffortBudgetEscalation(b EffortBudget) error {
	if !b.EscalationAllowed && b.MaxEscalations != 0 {
		return fmt.Errorf("effort budget: max_escalations must be zero when escalation is disabled")
	}
	if b.EscalationLevel < 0 {
		return fmt.Errorf("effort budget: escalation_level must not be negative")
	}
	if b.EscalationLevel > b.MaxEscalations {
		return fmt.Errorf("effort budget: escalation_level exceeds max_escalations")
	}
	return nil
}

func containsBudgetValue(values []string, value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

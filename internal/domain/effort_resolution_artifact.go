package domain

import (
	"fmt"
	"strings"
)

// EffortResolutionArtifactSchemaVersion identifies the versioned projection
// of one LEVELING decision. It is evidence of resolution, not a second policy
// authority and not a replacement for the append-only LEVELING ledger.
const EffortResolutionArtifactSchemaVersion = "strategist-effort-resolution/v1"

// EffortResolutionArtifact records the model, effort, provenance, and fallback
// state selected for a Role. Empty model/effort values are valid: they make an
// unresolved host-only observation explicit instead of inventing confidence.
type EffortResolutionArtifact struct {
	SchemaVersion   string        `json:"schema_version" yaml:"schema_version"`
	TaxonomyVersion string        `json:"taxonomy_version,omitempty" yaml:"taxonomy_version,omitempty"`
	Role            string        `json:"role" yaml:"role"`
	Provider        string        `json:"provider,omitempty" yaml:"provider,omitempty"`
	Model           string        `json:"model,omitempty" yaml:"model,omitempty"`
	Effort          string        `json:"effort,omitempty" yaml:"effort,omitempty"`
	Source          string        `json:"source,omitempty" yaml:"source,omitempty"`
	Capability      string        `json:"capability,omitempty" yaml:"capability,omitempty"`
	FallbackUsed    bool          `json:"fallback_used,omitempty" yaml:"fallback_used,omitempty"`
	FallbackReason  string        `json:"fallback_reason,omitempty" yaml:"fallback_reason,omitempty"`
	PolicyVersion   int           `json:"policy_version,omitempty" yaml:"policy_version,omitempty"`
	PolicyDigest    string        `json:"policy_digest,omitempty" yaml:"policy_digest,omitempty"`
	Budget          *EffortBudget `json:"effort_budget,omitempty" yaml:"effort_budget,omitempty"`
}

// Validate checks the artifact envelope and the closed provenance vocabulary.
func (a EffortResolutionArtifact) Validate() error {
	if a.SchemaVersion != EffortResolutionArtifactSchemaVersion {
		return fmt.Errorf("effort resolution artifact: unsupported schema_version %q", a.SchemaVersion)
	}
	if err := ValidateTaxonomyVersion(a.TaxonomyVersion); err != nil {
		return fmt.Errorf("effort resolution artifact: %w", err)
	}
	if err := validateArtifactFields(a); err != nil {
		return err
	}
	if a.Budget != nil {
		if err := a.Budget.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func validateArtifactFields(a EffortResolutionArtifact) error {
	if err := validateEffortRole(a.Role); err != nil {
		return err
	}
	if err := validateEffortValues(a.Effort, a.Source); err != nil {
		return err
	}
	if err := validateEffortPolicy(a.PolicyVersion, a.PolicyDigest); err != nil {
		return err
	}
	return validateEffortFallback(a.FallbackUsed, a.FallbackReason)
}

func validateEffortRole(role string) error {
	if strings.TrimSpace(role) == "" {
		return fmt.Errorf("effort resolution artifact: role is required")
	}
	return nil
}

func validateEffortValues(effort, source string) error {
	if effort != "" && !validEffortResolution(effort) {
		return fmt.Errorf("effort resolution artifact: unsupported effort %q", effort)
	}
	if source != "" && source != "host" && source != "policy" {
		return fmt.Errorf("effort resolution artifact: unsupported source %q", source)
	}
	return nil
}

func validateEffortPolicy(version int, digest string) error {
	if version < 0 {
		return fmt.Errorf("effort resolution artifact: policy_version must not be negative")
	}
	if version > 0 && strings.TrimSpace(digest) == "" {
		return fmt.Errorf("effort resolution artifact: policy_digest is required with policy_version")
	}
	return nil
}

func validateEffortFallback(used bool, reason string) error {
	if used && strings.TrimSpace(reason) == "" {
		return fmt.Errorf("effort resolution artifact: fallback_reason is required when fallback_used")
	}
	return nil
}

func validEffortResolution(effort string) bool {
	switch strings.ToLower(strings.TrimSpace(effort)) {
	case "none", "low", "medium", "high", "xhigh", "max":
		return true
	default:
		return false
	}
}

package domain

import (
	"fmt"
	"strings"
)

// RoleSlotMap cataloged/custom Weapons resolve through their authority files;
// native roles resolve through roles/<provider>.yaml. The type and structural
// validation live in persona_types.go, while this boundary remains guarded by
// the catalog-first convergence check.

// ActiveConfig is the structure of a standalone active.yaml template. Provider
// resolution treats the catalog as the canonical authority for provider facts.
type ActiveConfig struct {
	// The provider catalog is the canonical provider authority; active.yaml
	// records operator intent and never defines provider identity or capability.
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
	// ADR configures where accepted Opportunity Attack ADRs are materialized.
	ADR ADRConfig `yaml:"adr,omitempty"`
	// Runbooks configures the canonical root for accepted Runbook artifacts.
	Runbooks RunbookConfig `yaml:"runbooks,omitempty"`
}

// ADRConfig is the optional active.yaml adr block.
type ADRConfig struct {
	// CanonicalPath is the project-relative directory holding numbered ADRs.
	// Empty means the archived fallback under base_path.
	CanonicalPath string `yaml:"canonical_path,omitempty"`
}

// RunbookConfig is the optional active.yaml runbooks block.
type RunbookConfig struct {
	// CanonicalPath is the project-relative directory holding accepted Runbooks.
	// Empty means the legacy docs/runbooks compatibility location is used for reads.
	CanonicalPath string `yaml:"canonical_path,omitempty"`
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

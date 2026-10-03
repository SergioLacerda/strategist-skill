package domain

import (
	"fmt"
	"strings"
)

// StageResolutionArtifactSchemaVersion identifies the durable, auditable
// projection of one Stage decision. LegacyRoute is correlation only; Stage is
// the canonical value consumed by runtime boundaries.
const StageResolutionArtifactSchemaVersion = "strategist-stage-resolution/v1"

// StageResolutionArtifact is the versioned boundary artifact for resolving a
// legacy route into FULL, SHORT, or ROSTER. Trigger is intentionally required
// so a replay can explain what caused the resolution without consulting the
// original prompt or provider output.
type StageResolutionArtifact struct {
	SchemaVersion   string `json:"schema_version" yaml:"schema_version"`
	TaxonomyVersion string `json:"taxonomy_version,omitempty" yaml:"taxonomy_version,omitempty"`
	Stage           Stage  `json:"stage" yaml:"stage"`
	Trigger         string `json:"trigger" yaml:"trigger"`
	Role            string `json:"role,omitempty" yaml:"role,omitempty"`
	Feat            string `json:"feat,omitempty" yaml:"feat,omitempty"`
	PolicyVersion   string `json:"policy_version" yaml:"policy_version"`
	Reason          string `json:"reason" yaml:"reason"`
	LegacyRoute     string `json:"legacy_route" yaml:"legacy_route"`
}

// NewStageResolutionArtifact converts a validated resolution into its durable
// projection. The trigger is supplied by the boundary that observed the
// decision (for example, Scout's request category).
func NewStageResolutionArtifact(resolution StageResolution, trigger string) (StageResolutionArtifact, error) {
	if err := resolution.Validate(); err != nil {
		return StageResolutionArtifact{}, fmt.Errorf("stage resolution artifact: %w", err)
	}
	artifact := StageResolutionArtifact{
		SchemaVersion:   StageResolutionArtifactSchemaVersion,
		TaxonomyVersion: CanonicalTaxonomyVersion,
		Stage:           resolution.Stage,
		Trigger:         strings.TrimSpace(trigger),
		Role:            strings.TrimSpace(resolution.Role),
		Feat:            strings.TrimSpace(resolution.Feat),
		PolicyVersion:   strings.TrimSpace(resolution.PolicyVersion),
		Reason:          strings.TrimSpace(resolution.Reason),
		LegacyRoute:     strings.TrimSpace(resolution.LegacyRoute),
	}
	if err := artifact.Validate(); err != nil {
		return StageResolutionArtifact{}, err
	}
	return artifact, nil
}

// Resolution reconstructs the domain resolution after validating the
// artifact. It is useful to keep persistence and runtime consumption on the
// same typed contract instead of reparsing untrusted JSON fields separately.
func (a StageResolutionArtifact) Resolution() (StageResolution, error) {
	if err := a.Validate(); err != nil {
		return StageResolution{}, err
	}
	return StageResolution{
		SchemaVersion:   a.SchemaVersion,
		TaxonomyVersion: a.TaxonomyVersion,
		Stage:           a.Stage,
		LegacyRoute:     a.LegacyRoute,
		Role:            a.Role,
		Feat:            a.Feat,
		PolicyVersion:   a.PolicyVersion,
		Reason:          a.Reason,
	}, nil
}

// Validate checks the complete artifact envelope and all required audit
// fields. Role and Feat remain optional because not every route has a role or
// Feat context at the resolution boundary.
func (a StageResolutionArtifact) Validate() error {
	if a.SchemaVersion != StageResolutionArtifactSchemaVersion {
		return fmt.Errorf("stage resolution artifact: unsupported schema_version %q", a.SchemaVersion)
	}
	if err := ValidateTaxonomyVersion(a.TaxonomyVersion); err != nil {
		return fmt.Errorf("stage resolution artifact: %w", err)
	}
	if err := a.Stage.Validate(); err != nil {
		return fmt.Errorf("stage resolution artifact: %w", err)
	}
	for name, value := range map[string]string{
		"trigger": a.Trigger, "policy_version": a.PolicyVersion,
		"reason": a.Reason, "legacy_route": a.LegacyRoute,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("stage resolution artifact: %s is required", name)
		}
	}
	return nil
}

package domain

import (
	"fmt"
	"strings"
)

// CanonicalTaxonomyVersion identifies the versioned seven-family vocabulary.
const CanonicalTaxonomyVersion = "strategist-taxonomy/v1"

// TaxonomyFamily is the canonical public family of a Strategist entity.
// Routes are deliberately not a family: they are compatibility inputs used to
// resolve a governed Stage.
type TaxonomyFamily string

const (
	// TaxonomyRole identifies the Role family.
	TaxonomyRole TaxonomyFamily = "role"
	// TaxonomyWeapon identifies the Weapon family.
	TaxonomyWeapon TaxonomyFamily = "weapon"
	// TaxonomyFeat identifies the Feat family.
	TaxonomyFeat TaxonomyFamily = "feat"
	// TaxonomyTool identifies the Tool family.
	TaxonomyTool TaxonomyFamily = "tool"
	// TaxonomyMechanism identifies the Mechanism family.
	TaxonomyMechanism TaxonomyFamily = "mechanism"
	// TaxonomyStage identifies the Stage family.
	TaxonomyStage TaxonomyFamily = "stage"
	// TaxonomyArtifact identifies the Artifact family.
	TaxonomyArtifact TaxonomyFamily = "artifact"
)

// Validate rejects values outside the canonical seven-family vocabulary.
func (f TaxonomyFamily) Validate() error {
	switch f {
	case TaxonomyRole, TaxonomyWeapon, TaxonomyFeat, TaxonomyTool,
		TaxonomyMechanism, TaxonomyStage, TaxonomyArtifact:
		return nil
	default:
		return fmt.Errorf("taxonomy family %q is not canonical", f)
	}
}

// CanonicalIdentity is the stable family-aware identity of a public entity.
// Version is required for Weapons because a Role binding must never resolve a
// provider by unversioned name or catalog order.
type CanonicalIdentity struct {
	Family  TaxonomyFamily `json:"family" yaml:"family"`
	ID      string         `json:"id" yaml:"id"`
	Version string         `json:"version,omitempty" yaml:"version,omitempty"`
}

// CanonicalIdentity returns the family-aware identity of a Role.
func (r Role) CanonicalIdentity() CanonicalIdentity {
	return CanonicalIdentity{Family: TaxonomyRole, ID: r.ID}
}

// CanonicalIdentity returns the family-aware identity of a Weapon manifest.
func (w WeaponManifest) CanonicalIdentity() CanonicalIdentity {
	return CanonicalIdentity{Family: TaxonomyWeapon, ID: w.ID, Version: w.Version}
}

// CanonicalIdentity returns the family-aware identity of a compiled Weapon.
func (w CompiledWeapon) CanonicalIdentity() CanonicalIdentity {
	return CanonicalIdentity{Family: TaxonomyWeapon, ID: w.ID, Version: w.Version}
}

// Validate checks family-aware identity without inferring family from a path
// or a historical identifier.
func (i CanonicalIdentity) Validate() error {
	if err := i.Family.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(i.ID) == "" {
		return fmt.Errorf("taxonomy identity %s requires id", i.Family)
	}
	if i.Family == TaxonomyWeapon && strings.TrimSpace(i.Version) == "" {
		return fmt.Errorf("Weapon identity %q requires version", i.ID)
	}
	return nil
}

// Stage is a governed operational flow. Internal phases are not additional
// public stages.
type Stage string

const (
	// StageFull identifies the complete governed flow.
	StageFull Stage = "FULL"
	// StageShort identifies the bounded short flow.
	StageShort Stage = "SHORT"
	// StageRoster identifies the roster and installation flow.
	StageRoster Stage = "ROSTER"
)

// Validate rejects stages outside the canonical vocabulary.
func (s Stage) Validate() error {
	switch s {
	case StageFull, StageShort, StageRoster:
		return nil
	default:
		return fmt.Errorf("stage %q is not canonical", s)
	}
}

// StageResolutionRequest contains the explicit context needed to map a route
// compatibility value. MissionExecution is required before direct_execute can
// be represented as the bounded SHORT stage.
type StageResolutionRequest struct {
	Route            string
	Role             string
	Feat             string
	PolicyVersion    string
	MissionExecution bool
}

// StageResolution is the auditable result of resolving a route into a Stage.
// LegacyRoute is correlation only; it never becomes a second taxonomy.
type StageResolution struct {
	SchemaVersion   string `json:"schema_version" yaml:"schema_version"`
	TaxonomyVersion string `json:"taxonomy_version,omitempty" yaml:"taxonomy_version,omitempty"`
	Stage           Stage  `json:"stage" yaml:"stage"`
	LegacyRoute     string `json:"legacy_route,omitempty" yaml:"legacy_route,omitempty"`
	Role            string `json:"role,omitempty" yaml:"role,omitempty"`
	Feat            string `json:"feat,omitempty" yaml:"feat,omitempty"`
	PolicyVersion   string `json:"policy_version" yaml:"policy_version"`
	Reason          string `json:"reason" yaml:"reason"`
}

const stageResolutionSchemaVersion = "strategist-stage-resolution/v1"

// ResolveStage maps supported legacy route values to the canonical Stage set.
// Unknown or unsafe mappings fail closed instead of inventing a fourth Stage.
func ResolveStage(request StageResolutionRequest) (StageResolution, error) {
	policyVersion := strings.TrimSpace(request.PolicyVersion)
	if policyVersion == "" {
		policyVersion = "stage-resolution/v1"
	}
	legacyRoute := strings.TrimSpace(request.Route)
	if legacyRoute == "" {
		legacyRoute = MissionRouteFullPipeline
	}

	resolution := StageResolution{
		SchemaVersion:   stageResolutionSchemaVersion,
		TaxonomyVersion: CanonicalTaxonomyVersion,
		LegacyRoute:     legacyRoute,
		Role:            strings.TrimSpace(request.Role),
		Feat:            strings.TrimSpace(request.Feat),
		PolicyVersion:   policyVersion,
	}

	switch legacyRoute {
	case MissionRouteFullPipeline:
		resolution.Stage = StageFull
		resolution.Reason = "legacy route resolved to FULL"
	case "implementation_short_route":
		resolution.Stage = StageShort
		resolution.Reason = "legacy route resolved to bounded SHORT"
	case "critical_hit":
		resolution.Stage = StageShort
		if resolution.Feat == "" {
			resolution.Feat = "critical_hit"
		}
		resolution.Reason = "Critical Hit compatibility route bounded to SHORT"
	case MissionRouteDirectExecute:
		if !request.MissionExecution {
			return StageResolution{}, fmt.Errorf("route %q requires explicit mission execution context", legacyRoute)
		}
		resolution.Stage = StageShort
		resolution.Reason = "direct_execute compatibility route bounded to SHORT"
	case "roster":
		resolution.Stage = StageRoster
		resolution.Reason = "roster configuration resolved to ROSTER"
	default:
		return StageResolution{}, fmt.Errorf("route %q cannot be resolved to a canonical Stage", legacyRoute)
	}

	return resolution, nil
}

// Validate checks that a resolution is complete and auditable.
func (r StageResolution) Validate() error {
	if r.SchemaVersion == "" {
		return fmt.Errorf("stage resolution requires schema_version")
	}
	if err := ValidateTaxonomyVersion(r.TaxonomyVersion); err != nil {
		return fmt.Errorf("stage resolution: %w", err)
	}
	if err := r.Stage.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(r.PolicyVersion) == "" {
		return fmt.Errorf("stage resolution requires policy_version")
	}
	if strings.TrimSpace(r.Reason) == "" {
		return fmt.Errorf("stage resolution requires reason")
	}
	return nil
}

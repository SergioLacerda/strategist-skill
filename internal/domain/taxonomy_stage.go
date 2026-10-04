package domain

import (
	"fmt"
	"strings"
)

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

// StageRequest is the explicit request a Role submits when it needs the
// resolver to select an operational Stage. Route is a compatibility input;
// Stage is the canonical result and is never selected by a Feat autonomously.
// MissionExecution is required before direct_execute can be represented as the
// bounded SHORT stage.
type StageRequest struct {
	Route            string `json:"route,omitempty" yaml:"route,omitempty"`
	Role             string `json:"role,omitempty" yaml:"role,omitempty"`
	Feat             string `json:"feat,omitempty" yaml:"feat,omitempty"`
	PolicyVersion    string `json:"policy_version,omitempty" yaml:"policy_version,omitempty"`
	MissionID        string `json:"mission_id,omitempty" yaml:"mission_id,omitempty"`
	CorrelationKey   string `json:"correlation_key,omitempty" yaml:"correlation_key,omitempty"`
	GateRequired     bool   `json:"gate_required,omitempty" yaml:"gate_required,omitempty"`
	MissionExecution bool   `json:"mission_execution,omitempty" yaml:"mission_execution,omitempty"`
}

// StageResolutionRequest is retained as a source-compatible name while
// callers migrate to the taxonomy's Role-owned StageRequest vocabulary.
type StageResolutionRequest = StageRequest

// StageResolution is the auditable result of resolving a route into a Stage.
// LegacyRoute is correlation only; it never becomes a second taxonomy.
type StageResolution struct {
	SchemaVersion   string `json:"schema_version" yaml:"schema_version"`
	TaxonomyVersion string `json:"taxonomy_version,omitempty" yaml:"taxonomy_version,omitempty"`
	Stage           Stage  `json:"stage" yaml:"stage"`
	LegacyRoute     string `json:"legacy_route,omitempty" yaml:"legacy_route,omitempty"`
	Role            string `json:"role,omitempty" yaml:"role,omitempty"`
	Feat            string `json:"feat,omitempty" yaml:"feat,omitempty"`
	MissionID       string `json:"mission_id,omitempty" yaml:"mission_id,omitempty"`
	CorrelationKey  string `json:"correlation_key,omitempty" yaml:"correlation_key,omitempty"`
	GateRequired    bool   `json:"gate_required,omitempty" yaml:"gate_required,omitempty"`
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
		MissionID:       strings.TrimSpace(request.MissionID),
		CorrelationKey:  strings.TrimSpace(request.CorrelationKey),
		GateRequired:    request.GateRequired,
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

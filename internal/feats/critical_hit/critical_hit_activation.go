package criticalhit

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

const (
	// FeatID is the stable taxonomy identity. The selected route with the
	// same spelling is only a compatibility representation of this Feat.
	FeatID = "critical_hit"
	// ActivationSchemaVersion versions the passive Feat result boundary.
	ActivationSchemaVersion = "strategist-critical-hit-activation/v1"
	// ActivationPolicyVersion versions the deterministic, Mechanism-backed
	// eligibility rules.
	ActivationPolicyVersion = "critical-hit-eligibility/v1"
)

// ActivationRequest is contextual input supplied by the owning Role. The Feat
// consumes this value and returns data; it does not select, open, or execute a
// Stage and does not authorize repository mutation.
type ActivationRequest struct {
	MissionID string
	Role      string
	Evidence  Evidence
}

// ActivationResult is the passive Feat boundary. StageRequest is an explicit
// downstream request for the Role/resolver boundary; it is not an execution
// authorization. An ineligible result carries a FULL fallback request so an
// incomplete or malformed activation cannot enter the direct-execution regime.
type ActivationResult struct {
	SchemaVersion  string                   `json:"schema_version" yaml:"schema_version"`
	Identity       domain.CanonicalIdentity `json:"identity" yaml:"identity"`
	Feat           string                   `json:"feat" yaml:"feat"`
	MissionID      string                   `json:"mission_id" yaml:"mission_id"`
	Role           string                   `json:"role" yaml:"role"`
	CorrelationKey string                   `json:"correlation_key" yaml:"correlation_key"`
	Mode           Mode                     `json:"mode" yaml:"mode"`
	Allowed        bool                     `json:"allowed" yaml:"allowed"`
	Reason         string                   `json:"reason,omitempty" yaml:"reason,omitempty"`
	FallbackRoute  string                   `json:"fallback_route,omitempty" yaml:"fallback_route,omitempty"`
	GateRequired   bool                     `json:"gate_required" yaml:"gate_required"`
	StageRequest   domain.StageRequest      `json:"stage_request" yaml:"stage_request"`
}

// Validate prevents an untrusted or malformed result from being interpreted as
// execution authorization. The Feat may request only its compatibility route
// (which resolves to SHORT) or the conservative FULL fallback; direct_execute
// is never a valid Feat result.
func (r ActivationResult) Validate() error {
	if r.SchemaVersion != ActivationSchemaVersion {
		return fmt.Errorf("critical hit activation: unsupported schema_version %q", r.SchemaVersion)
	}
	if err := validateActivationIdentity(r); err != nil {
		return err
	}
	if err := validateActivationContext(r); err != nil {
		return err
	}
	if err := validateActivationStage(r); err != nil {
		return err
	}
	return nil
}

func validateActivationIdentity(r ActivationResult) error {
	if err := r.Identity.Validate(); err != nil {
		return fmt.Errorf("critical hit activation identity: %w", err)
	}
	if r.Identity.Family != domain.TaxonomyFeat || r.Identity.ID != FeatID || r.Feat != FeatID {
		return fmt.Errorf("critical hit activation requires Feat identity %q", FeatID)
	}
	return nil
}

func validateActivationContext(r ActivationResult) error {
	if strings.TrimSpace(r.MissionID) == "" || strings.TrimSpace(r.Role) == "" || strings.TrimSpace(r.CorrelationKey) == "" {
		return fmt.Errorf("critical hit activation requires mission, role, and correlation context")
	}
	if !r.GateRequired || !r.StageRequest.GateRequired {
		return fmt.Errorf("critical hit activation requires the approval gate")
	}
	if r.StageRequest.MissionExecution {
		return fmt.Errorf("critical hit activation cannot authorize mission execution")
	}
	if r.StageRequest.MissionID != r.MissionID || r.StageRequest.Role != r.Role || r.StageRequest.Feat != FeatID || r.StageRequest.CorrelationKey != r.CorrelationKey {
		return fmt.Errorf("critical hit activation StageRequest context does not match Feat context")
	}
	return nil
}

func validateActivationStage(r ActivationResult) error {
	resolution, err := domain.ResolveStage(r.StageRequest)
	if err != nil {
		return fmt.Errorf("critical hit activation StageRequest: %w", err)
	}
	wantStage, err := expectedActivationStage(r)
	if err != nil {
		return err
	}
	if resolution.Stage != wantStage {
		return fmt.Errorf("critical hit activation resolved to %s, want %s", resolution.Stage, wantStage)
	}
	return nil
}

func expectedActivationStage(r ActivationResult) (domain.Stage, error) {
	if r.Allowed {
		if r.StageRequest.Route != FeatID {
			return "", fmt.Errorf("eligible critical hit activation requires compatibility route %q", FeatID)
		}
		return domain.StageShort, nil
	}
	if r.StageRequest.Route != criticalHitFallbackFullPipeline {
		return "", fmt.Errorf("ineligible critical hit activation requires FULL fallback")
	}
	return domain.StageFull, nil
}

// FeatActivationRequest and FeatActivationResult are descriptive aliases for
// callers that want the taxonomy boundary explicit in their APIs.
type FeatActivationRequest = ActivationRequest

// FeatActivationResult is a descriptive alias for ActivationResult.
type FeatActivationResult = ActivationResult

// Activate evaluates Critical Hit in the Role-provided context and returns a
// typed Feat result plus a StageRequest. It performs no I/O or mutation.
func Activate(request ActivationRequest) (ActivationResult, error) {
	missionID := strings.TrimSpace(request.MissionID)
	role := strings.TrimSpace(request.Role)
	if missionID == "" {
		return ActivationResult{}, fmt.Errorf("critical hit activation requires mission_id")
	}
	if role == "" {
		return ActivationResult{}, fmt.Errorf("critical hit activation requires role")
	}

	decision := EvaluateEligibility(request.Evidence)
	correlationKey := missionID + ":" + FeatID
	stageRoute := criticalHitFallbackFullPipeline
	if decision.Allowed {
		stageRoute = FeatID
	}
	stageRequest := domain.StageRequest{
		Route:          stageRoute,
		Role:           role,
		Feat:           FeatID,
		PolicyVersion:  ActivationPolicyVersion,
		MissionID:      missionID,
		CorrelationKey: correlationKey,
		GateRequired:   true,
	}
	result := ActivationResult{
		SchemaVersion:  ActivationSchemaVersion,
		Identity:       domain.CanonicalIdentity{Family: domain.TaxonomyFeat, ID: FeatID},
		Feat:           FeatID,
		MissionID:      missionID,
		Role:           role,
		CorrelationKey: correlationKey,
		Mode:           decision.Mode,
		Allowed:        decision.Allowed,
		Reason:         decision.Reason,
		FallbackRoute:  decision.FallbackRoute,
		GateRequired:   true,
		StageRequest:   stageRequest,
	}
	if err := result.Validate(); err != nil {
		return ActivationResult{}, err
	}
	return result, nil
}

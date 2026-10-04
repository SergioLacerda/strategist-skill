package telemetry

import "github.com/SergioLacerda/strategist-skill/internal/domain"

const (
	// StageResolutionEventName identifies Stage resolution telemetry events.
	StageResolutionEventName = "strategist.stage.resolution"
	// StageResolutionContract identifies the Stage resolution event contract.
	StageResolutionContract = "stage-resolution/v1"
)

// NewStageResolutionEvent builds the canonical telemetry projection for a
// Stage decision. The legacy route remains a correlation attribute; runtime
// consumers should use Stage and the explicit trigger/policy fields.
func NewStageResolutionEvent(runID string, artifact domain.StageResolutionArtifact) Event {
	event := NewEvent(StageResolutionEventName, SeverityInfo, runID, true)
	event.Attributes = map[string]any{
		AttrEventContractID:    StageResolutionContract,
		AttrEventAuthority:     AuthorityStrategistLocal,
		AttrSchemaVersion:      artifact.SchemaVersion,
		AttrTaxonomyVersion:    domain.CanonicalTaxonomyVersion,
		AttrComponent:          "scout",
		AttrPhase:              "routing",
		AttrStatus:             "resolved",
		AttrStage:              string(artifact.Stage),
		AttrStageTrigger:       artifact.Trigger,
		AttrStagePolicyVersion: artifact.PolicyVersion,
		AttrStageReason:        artifact.Reason,
		AttrRoute:              artifact.LegacyRoute,
		AttrDeprecationState:   "compatibility_only",
		AttrDeprecatedValue:    artifact.LegacyRoute,
		AttrCanonicalValue:     string(artifact.Stage),
	}
	if artifact.Role != "" {
		event.Attributes[AttrRole] = artifact.Role
	}
	if artifact.Feat != "" {
		event.Attributes[AttrFeat] = artifact.Feat
	}
	if artifact.CorrelationKey != "" {
		event.Attributes[AttrCorrelationID] = artifact.CorrelationKey
	}
	if artifact.GateRequired {
		event.Attributes[AttrGateStatus] = "required"
	}
	return event
}

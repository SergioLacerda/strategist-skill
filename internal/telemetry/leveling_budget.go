package telemetry

import (
	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// LevelingBudgetEventKind separates policy recommendation from local
// enforcement and provider evidence. A local event never implies live provider
// execution.
type LevelingBudgetEventKind string

const (
	// LevelingBudgetRecommended identifies a policy recommendation event.
	LevelingBudgetRecommended LevelingBudgetEventKind = "recommendation"
	// LevelingBudgetGranted identifies a locally granted delegation.
	LevelingBudgetGranted LevelingBudgetEventKind = "grant"
	// LevelingBudgetConsumed identifies completed local consumption.
	LevelingBudgetConsumed LevelingBudgetEventKind = "consumption"
	// LevelingBudgetDenied identifies a locally denied request.
	LevelingBudgetDenied LevelingBudgetEventKind = "denial"
	// LevelingBudgetExhausted identifies a request denied by exhaustion.
	LevelingBudgetExhausted LevelingBudgetEventKind = "exhaustion"
	// LevelingBudgetEscalated identifies a new bounded escalation.
	LevelingBudgetEscalated LevelingBudgetEventKind = "escalation"
	// LevelingProviderUsage identifies provider usage evidence.
	LevelingProviderUsage LevelingBudgetEventKind = "provider_usage"
)

const (
	// LevelingBudgetRecommendedEvent is the recommendation event name.
	LevelingBudgetRecommendedEvent = "strategist.leveling.budget.recommendation"
	// LevelingBudgetGrantedEvent is the grant event name.
	LevelingBudgetGrantedEvent = "strategist.leveling.budget.grant"
	// LevelingBudgetConsumedEvent is the consumption event name.
	LevelingBudgetConsumedEvent = "strategist.leveling.budget.consumption"
	// LevelingBudgetDeniedEvent is the denial event name.
	LevelingBudgetDeniedEvent = "strategist.leveling.budget.denial"
	// LevelingBudgetExhaustedEvent is the exhaustion event name.
	LevelingBudgetExhaustedEvent = "strategist.leveling.budget.exhaustion"
	// LevelingBudgetEscalatedEvent is the escalation event name.
	LevelingBudgetEscalatedEvent = "strategist.leveling.budget.escalation"
	// LevelingProviderUsageEvent is the provider usage evidence event name.
	LevelingProviderUsageEvent = "strategist.leveling.provider.usage"
)

const (
	// AttrLevelingBudgetKind identifies the budget event kind.
	AttrLevelingBudgetKind = "strategist.leveling.budget.kind"
	// AttrLevelingBudgetSchemaVersion identifies the envelope schema.
	AttrLevelingBudgetSchemaVersion = "strategist.leveling.budget.schema_version"
	// AttrLevelingBudgetResolutionID identifies the resolved envelope.
	AttrLevelingBudgetResolutionID = "strategist.leveling.budget.resolution_id"
	// AttrLevelingBudgetParentResolution identifies the escalation parent.
	AttrLevelingBudgetParentResolution = "strategist.leveling.budget.parent_resolution_id"
	// AttrLevelingBudgetRemaining identifies unused delegation capacity.
	AttrLevelingBudgetRemaining = "strategist.leveling.budget.remaining"
	// AttrLevelingBudgetActive identifies active sequential consumption.
	AttrLevelingBudgetActive = "strategist.leveling.budget.active"
	// AttrLevelingBudgetContext identifies the context bound.
	AttrLevelingBudgetContext = "strategist.leveling.budget.context_budget"
	// AttrLevelingBudgetEvidenceDepth identifies the evidence bound.
	AttrLevelingBudgetEvidenceDepth = "strategist.leveling.budget.evidence_depth"
	// AttrLevelingProviderEvidence identifies live-provider evidence state.
	AttrLevelingProviderEvidence = "strategist.leveling.provider.evidence"
)

// LevelingBudgetEventInput carries correlation without requiring sinks to
// understand the domain envelope.
type LevelingBudgetEventInput struct {
	MissionID          string
	Role               string
	ResolutionID       string
	ParentResolutionID string
	Reason             string
	Remaining          int
	Active             int
	ProviderEvidence   string
}

// NewLevelingBudgetEvent builds one of the closed LEVELING budget events.
// Provider usage defaults to unverified unless an authorized live evidence
// source explicitly supplies another state.
func NewLevelingBudgetEvent(kind LevelingBudgetEventKind, runID string, budget domain.EffortBudget, input LevelingBudgetEventInput) Event {
	name, severity := levelingBudgetEventShape(kind)
	event := NewEvent(name, severity, runID, true)
	evidence := input.ProviderEvidence
	if evidence == "" {
		evidence = "unverified"
	}
	event.Attributes = map[string]any{
		AttrEventAuthority:                 AuthorityStrategistLocal,
		AttrEventDecision:                  string(kind),
		AttrLevelingBudgetKind:             string(kind),
		AttrLevelingBudgetSchemaVersion:    budget.SchemaVersion,
		AttrLevelingBudgetResolutionID:     input.ResolutionID,
		AttrLevelingBudgetParentResolution: input.ParentResolutionID,
		AttrLevelingBudgetRemaining:        input.Remaining,
		AttrLevelingBudgetActive:           input.Active,
		AttrMissionID:                      input.MissionID,
		AttrRole:                           input.Role,
		AttrReason:                         input.Reason,
		AttrProvider:                       budget.Provider,
		AttrModel:                          budget.Model,
		AttrLevelingBudgetContext:          budget.ContextBudget,
		AttrLevelingBudgetEvidenceDepth:    budget.EvidenceDepth,
		AttrLevelingPolicyVersion:          budget.PolicyVersion,
		AttrLevelingPolicyDigest:           budget.PolicyDigest,
		AttrLevelingProviderEvidence:       evidence,
	}
	return event
}

func levelingBudgetEventShape(kind LevelingBudgetEventKind) (string, SeverityNumber) {
	switch kind {
	case LevelingBudgetRecommended:
		return LevelingBudgetRecommendedEvent, SeverityInfo
	case LevelingBudgetGranted:
		return LevelingBudgetGrantedEvent, SeverityInfo
	case LevelingBudgetConsumed:
		return LevelingBudgetConsumedEvent, SeverityInfo
	case LevelingBudgetDenied:
		return LevelingBudgetDeniedEvent, SeverityWarn
	case LevelingBudgetExhausted:
		return LevelingBudgetExhaustedEvent, SeverityWarn
	case LevelingBudgetEscalated:
		return LevelingBudgetEscalatedEvent, SeverityInfo
	case LevelingProviderUsage:
		return LevelingProviderUsageEvent, SeverityInfo
	default:
		return "strategist.leveling.budget.unknown", SeverityWarn
	}
}

// Package missionview composes read-only mission authorities into a stable UX
// projection. It has no transition or persistence API.
package missionview

import (
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// SchemaVersion identifies the versioned machine-readable mission view.
const SchemaVersion = "strategist-mission-view/v1"

// Availability values make the absence of secondary sources explicit.
const (
	Available     = "available"
	Unavailable   = "unavailable"
	Unknown       = "unknown"
	NotApplicable = "not_applicable"
	// NoSample: the source was read but holds no calibrated sample yet, so
	// the numbers it would show carry no information.
	NoSample = "no_sample"
)

// Cataloged diagnostic reason codes. They describe only unavailable secondary
// read sources; none is an execution or authorization decision.
const (
	ReasonMissionViewConfidenceUnavailable     = "mission_view_confidence_unavailable"
	ReasonMissionViewGateOutcomeUnavailable    = "mission_view_gate_outcome_unavailable"
	ReasonMissionViewLevelingUnavailable       = "mission_view_leveling_unavailable"
	ReasonMissionViewTokenUsageUnavailable     = "mission_view_token_usage_unavailable"
	ReasonMissionViewHandoffMetricsUnavailable = "mission_view_handoff_metrics_unavailable"
)

// Input contains values already loaded from their respective authorities.
type Input struct {
	Status              domain.MissionEngineStatus
	Registry            domain.RoleRegistry
	SlotProviders       map[string]string
	Confidence          telemetry.ConfidenceGateReview
	ConfidenceError     error
	GateOutcome         string
	GateError           error
	Levels              []leveling.Record
	LevelsError         error
	Run                 string
	TokenUsage          []telemetry.MissionTokenUsageRecord
	TokenUsageError     error
	HandoffMetrics      []telemetry.RefinementHandoffLine
	HandoffMetricsError error
	DeclaredTokenBudget string
}

// View is the complete read-only mission projection.
type View struct {
	Schema       string            `json:"schema"`
	MissionID    string            `json:"mission_id"`
	Run          string            `json:"run,omitempty"`
	Lifecycle    Lifecycle         `json:"lifecycle"`
	Journey      []JourneyEntry    `json:"journey"`
	Confidence   ConfidenceSection `json:"confidence"`
	ApprovalGate GateSection       `json:"approval_gate"`
	Leveling     LevelingSection   `json:"leveling"`
	TokenUsage   TokenUsageSection `json:"token_usage"`
	Diagnostics  []Diagnostic      `json:"diagnostics"`
}

// Lifecycle is the authoritative mission engine state included in a View.
type Lifecycle struct {
	Phase             string `json:"phase"`
	State             string `json:"state"`
	HandoffAttempt    int    `json:"handoff_attempt,omitempty"`
	HandoffStatus     string `json:"handoff_status,omitempty"`
	HandoffNextAction string `json:"handoff_next_action,omitempty"`
}

// JourneyEntry is one role or the distinct Approval Gate step.
type JourneyEntry struct {
	Kind     string `json:"kind"`
	ID       string `json:"id"`
	Phase    int    `json:"phase,omitempty"`
	Provider string `json:"provider,omitempty"`
}

// ConfidenceSection reports advisory confidence evidence.
type ConfidenceSection struct {
	Availability string                          `json:"availability"`
	Advisory     bool                            `json:"advisory"`
	Review       *telemetry.ConfidenceGateReview `json:"review,omitempty"`
}

// GateSection reports the recorded human Approval Gate outcome.
type GateSection struct {
	Availability string `json:"availability"`
	Outcome      string `json:"outcome,omitempty"`
}

// LevelingSection contains effective and historical LEVELING records.
type LevelingSection struct {
	Availability string         `json:"availability"`
	Selection    string         `json:"selection"`
	Roles        []LevelingRole `json:"roles"`
}

// LevelingRole groups LEVELING records for one role.
type LevelingRole struct {
	Role      string            `json:"role"`
	Effective *leveling.Record  `json:"effective,omitempty"`
	History   []leveling.Record `json:"history,omitempty"`
}

// TokenUsageSection and buildTokenUsage live in token_usage.go.

// Diagnostic describes an unavailable secondary authority and its remedy.
type Diagnostic struct {
	Reason string `json:"reason"`
	Action string `json:"action"`
}

// Build composes loaded authorities without persisting or advancing any state.
func Build(in Input) View {
	v := View{
		Schema: SchemaVersion, MissionID: in.Status.MissionID, Run: in.Run,
		Lifecycle:    Lifecycle{Phase: string(in.Status.Phase), State: string(in.Status.State), HandoffAttempt: in.Status.HandoffAttempt, HandoffStatus: in.Status.HandoffStatus, HandoffNextAction: in.Status.HandoffNextAction},
		Journey:      buildJourney(in.Registry, in.SlotProviders),
		Confidence:   ConfidenceSection{Availability: confidenceAvailability(in.Confidence), Advisory: true, Review: &in.Confidence},
		ApprovalGate: GateSection{Availability: Available, Outcome: in.GateOutcome},
		Leveling:     buildLevels(in.Registry, in.Levels, in.Run),
		TokenUsage:   buildTokenUsage(in.TokenUsage, in.HandoffMetrics, in.DeclaredTokenBudget),
	}
	if in.ConfidenceError != nil {
		v.Confidence = ConfidenceSection{Availability: Unavailable, Advisory: true}
		v.Diagnostics = append(v.Diagnostics, Diagnostic{Reason: ReasonMissionViewConfidenceUnavailable, Action: "inspect confidence ledger"})
	}
	if in.GateError != nil {
		v.ApprovalGate = GateSection{Availability: Unavailable}
		v.Diagnostics = append(v.Diagnostics, Diagnostic{Reason: ReasonMissionViewGateOutcomeUnavailable, Action: "inspect ground-truth ledger"})
	} else if in.GateOutcome == "" {
		v.ApprovalGate.Availability = NotApplicable
	}
	if in.LevelsError != nil {
		v.Leveling = LevelingSection{Availability: Unavailable, Selection: selection(in.Run)}
		v.Diagnostics = append(v.Diagnostics, Diagnostic{Reason: ReasonMissionViewLevelingUnavailable, Action: "inspect role-level ledger"})
	}
	if in.TokenUsageError != nil {
		v.TokenUsage = TokenUsageSection{Availability: Unavailable, DeclaredTokenBudget: in.DeclaredTokenBudget}
		v.Diagnostics = append(v.Diagnostics, Diagnostic{Reason: ReasonMissionViewTokenUsageUnavailable, Action: "inspect mission-token-usage ledger"})
	}
	if in.HandoffMetricsError != nil {
		v.TokenUsage.LedgerComparison = telemetry.TokenLedgerComparison{Status: telemetry.TokenLedgerReportedOnly}
		v.Diagnostics = append(v.Diagnostics, Diagnostic{Reason: ReasonMissionViewHandoffMetricsUnavailable, Action: "inspect handoff-metrics ledger"})
	}
	return v
}

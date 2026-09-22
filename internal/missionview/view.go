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
	ReasonMissionViewConfidenceUnavailable  = "mission_view_confidence_unavailable"
	ReasonMissionViewGateOutcomeUnavailable = "mission_view_gate_outcome_unavailable"
	ReasonMissionViewLevelingUnavailable    = "mission_view_leveling_unavailable"
)

// Input contains values already loaded from their respective authorities.
type Input struct {
	Status          domain.MissionEngineStatus
	Registry        domain.RoleRegistry
	SlotProviders   map[string]string
	Confidence      telemetry.ConfidenceGateReview
	ConfidenceError error
	GateOutcome     string
	GateError       error
	Levels          []leveling.Record
	LevelsError     error
	Run             string
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
	return v
}

func buildJourney(reg domain.RoleRegistry, providers map[string]string) []JourneyEntry {
	roles := reg.Roles()
	out := make([]JourneyEntry, 0, len(roles)+1)
	for _, role := range roles {
		if phase, ok := reg.PhaseOf("gate"); ok && role.Phase == phase+1 {
			out = append(out, JourneyEntry{Kind: "gate", ID: "gate", Phase: phase})
		}
		entry := JourneyEntry{Kind: "role", ID: role.ID, Phase: role.Phase}
		if role.Slot != "" {
			entry.Provider = providers[role.Slot]
		}
		out = append(out, entry)
	}
	return out
}

func buildLevels(reg domain.RoleRegistry, records []leveling.Record, run string) LevelingSection {
	byRole := groupLevelRecords(records)
	section := LevelingSection{Availability: Available, Selection: selection(run)}
	for _, role := range reg.Roles() {
		section.Roles = append(section.Roles, levelRole(role.ID, byRole[role.ID], run))
	}
	// The ledger was readable but holds nothing for this mission: every role
	// is unresolved, which is not the same as "available".
	if len(records) == 0 {
		section.Availability = Unknown
	}
	return section
}

func groupLevelRecords(records []leveling.Record) map[string][]leveling.Record {
	byRole := make(map[string][]leveling.Record)
	for _, record := range records {
		byRole[record.Role] = append(byRole[record.Role], record)
	}
	return byRole
}

func levelRole(role string, records []leveling.Record, run string) LevelingRole {
	history := append([]leveling.Record(nil), records...)
	return LevelingRole{Role: role, History: history, Effective: effectiveRecord(history, run)}
}

func effectiveRecord(records []leveling.Record, run string) *leveling.Record {
	for i := len(records) - 1; i >= 0; i-- {
		if run == "" || records[i].Run == run {
			record := records[i]
			return &record
		}
	}
	return nil
}

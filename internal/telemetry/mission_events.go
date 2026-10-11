package telemetry

import (
	"fmt"
	"time"
)

// Contract ids and event names of the mission lifecycle and usage events.
// They follow the strategist.<area>.<event> convention of the envelope; the
// full record travels as attributes so a consumer can rebuild the ledger
// from a valid event stream.
const (
	MissionLifecycleContractID = "mission-lifecycle/v1"
	MissionUsageContractID     = "mission-usage/v1"

	EventMissionStarted      = "strategist.mission.started"
	EventMissionEnded        = "strategist.mission.ended"
	EventMissionPhaseStarted = "strategist.mission.phase.started"
	EventMissionPhaseEnded   = "strategist.mission.phase.ended"
	EventUsageRecorded       = "strategist.usage.recorded"
)

// Attribute keys of the lifecycle and usage events.
const (
	AttrLifecycleMarkerID   = "strategist.lifecycle.marker_id"
	AttrLifecyclePhaseRunID = "strategist.lifecycle.phase_run_id"
	AttrLifecycleEndReason  = "strategist.lifecycle.end_reason"
	AttrLifecycleStage      = "strategist.lifecycle.stage"
	AttrLifecycleObservedAt = "strategist.lifecycle.observed_at"
	AttrLifecycleSchema     = "strategist.lifecycle.schema_version"

	AttrUsageRecordID        = "strategist.usage.record_id"
	AttrUsageEventID         = "strategist.usage.event_id"
	AttrUsagePhaseRunID      = "strategist.usage.phase_run_id"
	AttrUsageRole            = "strategist.usage.role"
	AttrUsageInvocationID    = "strategist.usage.invocation_id"
	AttrUsageAttempt         = "strategist.usage.attempt"
	AttrUsageScope           = "strategist.usage.scope"
	AttrUsageSessionID       = "strategist.usage.session_id"
	AttrUsageRecordKind      = "strategist.usage.record_kind"
	AttrUsageTokensInput     = "strategist.usage.tokens_input"
	AttrUsageTokensOutput    = "strategist.usage.tokens_output"
	AttrUsageTokensCache     = "strategist.usage.tokens_cache"
	AttrUsageTokensReasoning = "strategist.usage.tokens_reasoning"
	AttrUsageProvider        = "strategist.usage.provider"
	AttrUsageModel           = "strategist.usage.model"
	AttrUsageSource          = "strategist.usage.measurement_source"
	AttrUsageQuality         = "strategist.usage.measurement_quality"
	AttrUsageDedupMissing    = "strategist.usage.dedup_unavailable"
	AttrUsageCalls           = "strategist.usage.calls"
	AttrUsageObservedAt      = "strategist.usage.observed_at"
	AttrUsageSchema          = "strategist.usage.schema_version"
)

var lifecycleEventNames = map[string]string{
	"mission_started": EventMissionStarted,
	"mission_ended":   EventMissionEnded,
	"phase_started":   EventMissionPhaseStarted,
	"phase_ended":     EventMissionPhaseEnded,
}

var lifecycleKindsByEvent = map[string]string{
	EventMissionStarted:      "mission_started",
	EventMissionEnded:        "mission_ended",
	EventMissionPhaseStarted: "phase_started",
	EventMissionPhaseEnded:   "phase_ended",
}

// NewMissionLifecycleEvent builds the event of one lifecycle marker. The
// event timestamp is the marker's own instant, so the stream replays the
// ledger faithfully; the correlation id is the mission id.
func NewMissionLifecycleEvent(rec MissionLifecycleRecord) (Event, error) {
	if err := ValidateMissionLifecycleRecord(rec); err != nil {
		return Event{}, fmt.Errorf("lifecycle event: %w", err)
	}
	at, err := time.Parse(time.RFC3339, rec.ObservedAt)
	if err != nil {
		return Event{}, fmt.Errorf("lifecycle event: observed_at: %w", err)
	}
	event := NewEvent(lifecycleEventNames[rec.MarkerKind], SeverityInfo, rec.MissionID, true)
	event.Timestamp = at.UTC()
	event.Attributes = map[string]any{
		AttrMissionID: rec.MissionID, AttrCorrelationID: rec.MissionID, AttrPhase: rec.Phase,
		AttrEventContractID: MissionLifecycleContractID, AttrEventAuthority: AuthorityStrategistLocal,
		AttrLifecycleMarkerID: rec.MarkerID, AttrLifecycleObservedAt: rec.ObservedAt, AttrLifecycleSchema: rec.SchemaVersion,
	}
	setIfNotEmpty(event.Attributes, AttrLifecyclePhaseRunID, rec.PhaseRunID)
	setIfNotEmpty(event.Attributes, AttrLifecycleEndReason, rec.EndReason)
	setIfNotEmpty(event.Attributes, AttrLifecycleStage, rec.Stage)
	return event, nil
}

// NewMissionUsageEvent builds the event of one usage observation. A count that
// was not observed is an absent attribute, never zero. No prompt, response or
// conversation content has a field here.
func NewMissionUsageEvent(rec MissionUsageRecord) Event {
	event := NewEvent(EventUsageRecorded, SeverityInfo, rec.MissionID, true)
	if at, err := time.Parse(time.RFC3339Nano, rec.ObservedAt); err == nil {
		event.Timestamp = at.UTC()
	}
	event.Attributes = map[string]any{
		AttrMissionID: rec.MissionID, AttrCorrelationID: rec.MissionID, AttrPhase: rec.Phase,
		AttrEventContractID: MissionUsageContractID, AttrEventAuthority: AuthorityStrategistLocal,
		AttrUsageRecordID: rec.RecordID, AttrUsageRole: rec.Role, AttrUsageAttempt: int64(rec.Attempt),
		AttrUsageScope: rec.UsageScope, AttrUsageRecordKind: rec.RecordKind,
		AttrUsageSource: rec.MeasurementSource, AttrUsageQuality: rec.MeasurementQuality,
		AttrUsageObservedAt: rec.ObservedAt, AttrUsageSchema: rec.SchemaVersion,
	}
	for key, value := range map[string]string{
		AttrUsageEventID: rec.EventID, AttrUsagePhaseRunID: rec.PhaseRunID, AttrUsageInvocationID: rec.InvocationID,
		AttrUsageSessionID: rec.SessionID, AttrUsageProvider: rec.Provider, AttrUsageModel: rec.Model,
	} {
		setIfNotEmpty(event.Attributes, key, value)
	}
	for key, value := range map[string]*int64{
		AttrUsageTokensInput: rec.TokensInput, AttrUsageTokensOutput: rec.TokensOutput,
		AttrUsageTokensCache: rec.TokensCache, AttrUsageTokensReasoning: rec.TokensReasoning,
	} {
		if value != nil {
			event.Attributes[key] = *value
		}
	}
	if rec.DedupUnavailable {
		event.Attributes[AttrUsageDedupMissing] = true
	}
	if rec.Calls > 0 {
		event.Attributes[AttrUsageCalls] = int64(rec.Calls)
	}
	return event
}

func setIfNotEmpty(m map[string]any, key, value string) {
	if value != "" {
		m[key] = value
	}
}

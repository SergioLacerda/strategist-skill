package domain

// LifecycleMarkerKind names one mission lifecycle marker. Markers are chronological
// milestones derived from engine transitions; they carry no token data.
type LifecycleMarkerKind string

// Lifecycle marker kinds.
const (
	MarkerMissionStarted LifecycleMarkerKind = "mission_started"
	MarkerPhaseStarted   LifecycleMarkerKind = "phase_started"
	MarkerPhaseEnded     LifecycleMarkerKind = "phase_ended"
	MarkerMissionEnded   LifecycleMarkerKind = "mission_ended"
)

// LifecycleEndReason explains why a phase or the mission ended.
type LifecycleEndReason string

// Lifecycle end reasons.
const (
	EndCompleted         LifecycleEndReason = "completed"
	EndRevisionRequested LifecycleEndReason = "revision_requested"
	EndBlocked           LifecycleEndReason = "blocked"
	EndRejected          LifecycleEndReason = "rejected"
	EndTimedOut          LifecycleEndReason = "timed_out"
	EndAnalysisDelivered LifecycleEndReason = "analysis_delivered"
	EndADRClosed         LifecycleEndReason = "adr_closed"
)

// LifecycleMarker is one marker to record. It has no run identifier and no
// timestamp: the application layer numbers runs and the adapter owns the clock.
type LifecycleMarker struct {
	Kind      LifecycleMarkerKind
	Phase     PipelinePhase
	EndReason LifecycleEndReason
	Stage     Stage
}

// phaseEndReasons maps the events that end a phase for a reason other than
// ordinary completion.
var phaseEndReasons = map[MissionEngineEvent]LifecycleEndReason{
	MissionEventGateRevision:              EndRevisionRequested,
	MissionEventHandoffFailed:             EndRevisionRequested,
	MissionEventRefinementArtifactInvalid: EndRevisionRequested,
	MissionEventGateDenied:                EndRejected,
	MissionEventCriticalHitGateDeclined:   EndRejected,
	MissionEventGateTimeout:               EndTimedOut,
	MissionEventNoTasks:                   EndAnalysisDelivered,
	MissionEventGateApprovedAnalysisOnly:  EndAnalysisDelivered,
	MissionEventHandoffNotApplicable:      EndAnalysisDelivered,
}

// LifecycleMarkersForStart returns the markers written when a mission is created.
func LifecycleMarkersForStart(status MissionEngineStatus) []LifecycleMarker {
	return []LifecycleMarker{
		{Kind: MarkerMissionStarted, Phase: status.Phase, Stage: status.Stage},
		{Kind: MarkerPhaseStarted, Phase: status.Phase, Stage: status.Stage},
	}
}

// LifecycleMarkersForTransition derives the markers of one engine transition
// from the status before and after the event. A transition that keeps the
// canonical phase (retries, the handoff challenge inside the Approval Gate)
// produces none. It is pure: no clock, no run numbering, no IO.
func LifecycleMarkersForTransition(event MissionEngineEvent, before, after MissionEngineStatus) []LifecycleMarker {
	if before.Phase == after.Phase {
		return nil
	}
	var markers []LifecycleMarker
	if before.Phase != PhaseDone && before.Phase != PhaseBlocked {
		markers = append(markers, LifecycleMarker{
			Kind: MarkerPhaseEnded, Phase: before.Phase, Stage: after.Stage,
			EndReason: phaseEndReason(event, after),
		})
	}
	if after.Phase == PhaseBlocked {
		return markers // BLOCKED is terminal and is not a started phase
	}
	if after.Phase == PhaseDone {
		return append(markers, LifecycleMarker{
			Kind: MarkerMissionEnded, Phase: PhaseDone, Stage: after.Stage,
			EndReason: missionEndReason(event, after),
		})
	}
	markers = append(markers, LifecycleMarker{Kind: MarkerPhaseStarted, Phase: after.Phase, Stage: after.Stage})
	return markers
}

func phaseEndReason(event MissionEngineEvent, after MissionEngineStatus) LifecycleEndReason {
	if after.Phase == PhaseBlocked {
		return EndBlocked
	}
	if reason, ok := phaseEndReasons[event]; ok {
		return reason
	}
	return EndCompleted
}

func missionEndReason(event MissionEngineEvent, after MissionEngineStatus) LifecycleEndReason {
	if after.State == StateADRDone {
		return EndADRClosed
	}
	if after.State == StateDoneDelivery || after.State == StateDirectDone {
		return EndCompleted
	}
	if reason, ok := phaseEndReasons[event]; ok {
		return reason
	}
	return EndAnalysisDelivered
}

package telemetry

// Ranger-to-Archivist handoff events expose only bounded lifecycle facts. The
// artifact body, challenge answers, and artifact digest remain out of the
// telemetry envelope.
const (
	RangerToArchivistEventName  = "strategist.handoff.ranger_to_archivist"
	RangerToArchivistContractID = "ranger-to-archivist-handoff/v1"
	AttrHandoffOutcome          = "strategist.handoff.outcome"
	AttrHandoffRequired         = "strategist.handoff.required"
	AttrHandoffAttempt          = "strategist.handoff.attempt"
)

// NewRangerToArchivistEvent builds the lifecycle-owned handoff event. A
// blocked/indeterminate result is an error event and carries only a stable
// reason token, never raw artifact or challenge content.
func NewRangerToArchivistEvent(runID, outcome string, required bool, attempt int, status string, criticalFailures int, reason string) Event {
	severity := SeverityInfo
	completeStatus := "done"
	if outcome == "failed" || outcome == "indeterminate" {
		severity = SeverityError
		completeStatus = "blocked"
	}
	event := NewEvent(RangerToArchivistEventName, severity, runID, true)
	event.Attributes = map[string]any{
		AttrEventContractID:                  RangerToArchivistContractID,
		AttrEventAuthority:                   AuthorityStrategistLocal,
		AttrComponent:                        "handoff",
		AttrPhase:                            "refinement",
		AttrTransitionGroup:                  "ranger_to_archivist",
		AttrStatus:                           completeStatus,
		AttrHandoffOutcome:                   outcome,
		AttrHandoffRequired:                  required,
		AttrHandoffAttempt:                   attempt,
		AttrHandoffChallengeStatus:           status,
		AttrHandoffChallengeCriticalFailures: criticalFailures,
	}
	if reason != "" {
		event.Attributes[AttrReason] = reason
	}
	return event
}

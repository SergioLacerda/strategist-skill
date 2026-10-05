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
	return newHandoffEvent(RangerToArchivistEventName, RangerToArchivistContractID, "refinement", "ranger_to_archivist", runID, outcome, required, attempt, status, criticalFailures, reason)
}

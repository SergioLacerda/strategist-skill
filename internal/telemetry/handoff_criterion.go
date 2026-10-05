package telemetry

// Archivist-to-Sniper handoff events expose only bounded lifecycle facts, like
// the Ranger-to-Archivist event: the artifact body, challenge answers and
// digests stay out of the telemetry envelope.
const (
	ArchivistToSniperEventName  = "strategist.handoff.archivist_to_sniper"
	ArchivistToSniperContractID = "archivist-to-sniper-handoff/v1"
)

// Criterion attributes say how a handoff was judged: by the deterministic main
// path, or with a provider reading. All are content-free.
const (
	AttrHandoffCriterionSource           = "strategist.handoff.criterion_source"
	AttrHandoffProvider                  = "strategist.handoff.provider"
	AttrHandoffProviderModel             = "strategist.handoff.provider_model"
	AttrHandoffProviderConfidencePercent = "strategist.handoff.provider_confidence_percent"
	AttrHandoffProviderThresholdPercent  = "strategist.handoff.provider_threshold_percent"
	AttrHandoffProviderApproved          = "strategist.handoff.provider_approved"

	CriterionSourceMain     = "main"
	CriterionSourceProvider = "provider"
)

// HandoffCriterion is the provider reading behind a handoff outcome. The zero
// value means no provider took part: the main path's deterministic checks did.
type HandoffCriterion struct {
	Provider          string
	Model             string
	ConfidencePercent int
	ThresholdPercent  int
	Approved          bool
}

// WithHandoffCriterion adds the criterion attributes to a handoff event.
func WithHandoffCriterion(event Event, criterion HandoffCriterion) Event {
	if event.Attributes == nil {
		event.Attributes = map[string]any{}
	}
	if criterion.Provider == "" {
		event.Attributes[AttrHandoffCriterionSource] = CriterionSourceMain
		return event
	}
	event.Attributes[AttrHandoffCriterionSource] = CriterionSourceProvider
	event.Attributes[AttrHandoffProvider] = criterion.Provider
	event.Attributes[AttrHandoffProviderModel] = criterion.Model
	event.Attributes[AttrHandoffProviderConfidencePercent] = criterion.ConfidencePercent
	event.Attributes[AttrHandoffProviderThresholdPercent] = criterion.ThresholdPercent
	event.Attributes[AttrHandoffProviderApproved] = criterion.Approved
	return event
}

// NewArchivistToSniperEvent builds the lifecycle-owned Archivist-to-Sniper
// handoff event. A failed result is an error event carrying only a stable reason.
func NewArchivistToSniperEvent(runID, outcome string, required bool, attempt int, status string, criticalFailures int, reason string) Event {
	return newHandoffEvent(ArchivistToSniperEventName, ArchivistToSniperContractID, "execution", "archivist_to_sniper", runID, outcome, required, attempt, status, criticalFailures, reason)
}

// newHandoffEvent is the one shape of a lifecycle handoff event: bounded
// lifecycle facts and a stable reason, never artifact content.
func newHandoffEvent(name, contractID, phase, group, runID, outcome string, required bool, attempt int, status string, criticalFailures int, reason string) Event {
	severity := SeverityInfo
	completeStatus := "done"
	if outcome == "failed" || outcome == "indeterminate" {
		severity = SeverityError
		completeStatus = "blocked"
	}
	event := NewEvent(name, severity, runID, true)
	event.Attributes = map[string]any{
		AttrEventContractID:                  contractID,
		AttrEventAuthority:                   AuthorityStrategistLocal,
		AttrComponent:                        "handoff",
		AttrPhase:                            phase,
		AttrTransitionGroup:                  group,
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

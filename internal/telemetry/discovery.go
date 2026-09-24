package telemetry

// Discovery weapon event names and values are intentionally closed: consumers
// must be able to distinguish static readiness from a live invocation and from
// Ranger's normalization result.
const (
	DiscoveryWeaponEventName = "strategist.discovery.weapon_invocation"

	DiscoveryInvocationInvoked = "invoked"
	DiscoveryInvocationFailed  = "failed"

	DiscoveryNormalizationNormalized   = "normalized"
	DiscoveryNormalizationRejected     = "rejected"
	DiscoveryNormalizationNotAttempted = "not_attempted"

	DiscoveryWeaponContractID = "ranger-discovery-weapon/v1"

	AttrDiscoveryInvocationStatus = "strategist.discovery.invocation_status"
	AttrDiscoveryNormalization    = "strategist.discovery.normalization_status"
	AttrInvocationEvidence        = "strategist.discovery.invocation_evidence"
)

// NewDiscoveryWeaponEvent builds the auditable Ranger boundary event. The
// provider payload is never included; only trusted invocation evidence and
// stable outcome categories are recorded.
func NewDiscoveryWeaponEvent(runID, provider, artifactPath, invocationStatus, normalizationStatus, evidence, reason string) Event {
	failed := invocationStatus != DiscoveryInvocationInvoked || normalizationStatus != DiscoveryNormalizationNormalized
	severity := SeverityInfo
	status := "done"
	if failed {
		severity = SeverityError
		status = "blocked"
	}
	event := NewEvent(DiscoveryWeaponEventName, severity, runID, true)
	event.Attributes = map[string]any{
		AttrEventContractID:           DiscoveryWeaponContractID,
		AttrEventAuthority:            AuthorityStrategistLocal,
		AttrComponent:                 "ranger",
		AttrPhase:                     "discovery",
		AttrRole:                      "ranger",
		AttrProvider:                  provider,
		AttrSelectedSkill:             provider,
		AttrArtifactPath:              SanitizePath(artifactPath),
		AttrStatus:                    status,
		AttrDiscoveryInvocationStatus: invocationStatus,
		AttrDiscoveryNormalization:    normalizationStatus,
	}
	if evidence != "" {
		event.Attributes[AttrInvocationEvidence] = evidence
	}
	if reason != "" {
		event.Attributes[AttrReason] = reason
	}
	return event
}

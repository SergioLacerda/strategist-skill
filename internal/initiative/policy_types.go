package initiative

// AbilityName identifies the INITIATIVE ability.
const AbilityName = "INITIATIVE"

// EffortTier is the reasoning effort level a role is expected to apply.
type EffortTier string

const (
	// EffortLow is the lowest reasoning effort tier.
	EffortLow EffortTier = "low"
	// EffortMedium is the default reasoning effort tier.
	EffortMedium EffortTier = "medium"
	// EffortHigh is an elevated reasoning effort tier.
	EffortHigh EffortTier = "high"
	// EffortXHigh is the highest reasoning effort tier.
	EffortXHigh EffortTier = "xhigh"
)

// AlignmentState describes how an observed effort compares to the
// recommended one.
type AlignmentState string

const (
	// AlignmentMatched means the observed effort matches the recommendation.
	AlignmentMatched AlignmentState = "matched"
	// AlignmentBelowRecommendation means the observed effort is lower than
	// recommended.
	AlignmentBelowRecommendation AlignmentState = "below_recommendation"
	// AlignmentAboveRecommendation means the observed effort is higher than
	// recommended.
	AlignmentAboveRecommendation AlignmentState = "above_recommendation"
	// AlignmentUnknown means the observed effort could not be determined.
	AlignmentUnknown AlignmentState = "unknown"
	// AlignmentUnavailable means no observation was available.
	AlignmentUnavailable AlignmentState = "unavailable"
	// AlignmentNotComparable means the observed and recommended efforts
	// cannot be ranked against each other.
	AlignmentNotComparable AlignmentState = "not_comparable"
)

// ObservationState describes the availability of an Observation.
type ObservationState string

const (
	// ObservationKnown means the observation was resolved.
	ObservationKnown ObservationState = "known"
	// ObservationUnknown means the observation could not be resolved.
	ObservationUnknown ObservationState = "unknown"
	// ObservationUnavailable means no observation was attempted.
	ObservationUnavailable ObservationState = "unavailable"
	// ObservationNotComparable means the observation exists but cannot be
	// ranked.
	ObservationNotComparable ObservationState = "not_comparable"
)

// Trigger identifies why an Advice was produced or re-evaluated.
type Trigger string

const (
	// TriggerInitial marks the first advice issued for a mission run.
	TriggerInitial Trigger = "initial"
	// TriggerScopeChanged marks re-evaluation after the mission scope changed.
	TriggerScopeChanged Trigger = "scope_changed"
	// TriggerSecurityRiskDiscovered marks re-evaluation after a security risk
	// was discovered.
	TriggerSecurityRiskDiscovered Trigger = "security_risk_discovered"
	// TriggerConflictingEvidence marks re-evaluation after conflicting
	// evidence was found.
	TriggerConflictingEvidence Trigger = "conflicting_evidence"
	// TriggerHandoffChallenged marks re-evaluation after a handoff was
	// challenged.
	TriggerHandoffChallenged Trigger = "handoff_challenged"
	// TriggerRepeatedFailure marks re-evaluation after repeated failures.
	TriggerRepeatedFailure Trigger = "repeated_failure"
	// TriggerUserRevisionRequested marks re-evaluation after the user
	// requested a revision.
	TriggerUserRevisionRequested Trigger = "user_revision_requested"
	// TriggerMandatoryObligationBlocked marks re-evaluation after a
	// mandatory obligation was blocked.
	TriggerMandatoryObligationBlocked Trigger = "mandatory_obligation_blocked"
)

var validTriggers = map[Trigger]bool{
	TriggerInitial: true, TriggerScopeChanged: true,
	TriggerSecurityRiskDiscovered: true, TriggerConflictingEvidence: true,
	TriggerHandoffChallenged: true, TriggerRepeatedFailure: true,
	TriggerUserRevisionRequested: true, TriggerMandatoryObligationBlocked: true,
}

// Profile is the per-role advice configuration within a Policy.
type Profile struct {
	RecommendedCapability string     `yaml:"recommended_capability" json:"recommended_capability"`
	RecommendedEffort     EffortTier `yaml:"recommended_effort" json:"recommended_effort"`
	Diligence             []string   `yaml:"diligence" json:"diligence"`
	ConfidenceCeiling     string     `yaml:"confidence_ceiling" json:"confidence_ceiling"`
}

// Policy is the standalone INITIATIVE configuration mapping roles to
// Profiles.
type Policy struct {
	Version  string             `yaml:"version" json:"version"`
	Profiles map[string]Profile `yaml:"profiles" json:"profiles"`
	Triggers []Trigger          `yaml:"reevaluation_triggers" json:"reevaluation_triggers"`
}

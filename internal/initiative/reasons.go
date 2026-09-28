package initiative

// Stable diagnostic reason codes for semantic and persistence validation.
const (
	ReasonPolicyInvalid          = "initiative_policy_invalid"
	ReasonConfidenceTierInvalid  = "initiative_confidence_tier_invalid"
	ReasonObligationInvalid      = "initiative_obligation_invalid"
	ReasonEvidenceInvalid        = "initiative_evidence_invalid"
	ReasonResultRevisionConflict = "initiative_result_revision_conflict"
	ReasonRecordInvalid          = "initiative_record_invalid"
	ReasonLedgerRecordOversized  = "initiative_ledger_record_oversized"

	// PRECISE-SHOT assessment reasons are closed and stable. Obligation-specific
	// reasons append ":<id>" without changing the base vocabulary.
	PreciseReasonBlockedObligation         = "blocked_obligation"
	PreciseReasonPartialObligation         = "partial_obligation"
	PreciseReasonMissingEvidence           = "missing_result_evidence"
	PreciseReasonInvalidEvidence           = "invalid_evidence"
	PreciseReasonStaleEvidence             = "stale_evidence"
	PreciseReasonUnverifiedEvidence        = "unverified_evidence"
	PreciseReasonConflictingEvidence       = "conflicting_evidence"
	PreciseReasonEffortBelowRecommendation = "effort_below_recommendation"
	PreciseReasonMaterialScopeChange       = "material_scope_change"
	PreciseReasonRepeatedFailure           = "repeated_failure"
	PreciseReasonCriticalSecurityRisk      = "critical_security_risk"
	PreciseReasonConfidenceCeilingApplied  = "confidence_ceiling_applied"
	PreciseReasonDeviation                 = "deviation"
	PreciseReasonNegativeOutcome           = "negative_outcome"
)

package leveling

// Stable diagnostic reason codes emitted by LEVELING validation and authority
// checks. They are catalogued as machine-enforced because callers return these
// errors on their reachable activation paths.
const (
	ReasonLevelingLedgerRecordOversized     = "leveling_ledger_record_oversized"
	ReasonLevelingLockUnavailable           = "leveling_ledger_lock_unavailable"
	ReasonLevelingPolicyStale               = "leveling_policy_stale"
	ReasonLevelingPolicyDigestMismatch      = "leveling_policy_digest_mismatch"
	ReasonLevelingPolicyMissing             = "leveling_policy_missing"
	ReasonLevelingPolicyUnreadable          = "leveling_policy_unreadable"
	ReasonLevelingRankedProviderIneligible  = "leveling_ranked_provider_ineligible"
	ReasonLevelingProviderUnknown           = "leveling_provider_unknown"
	ReasonLevelingProviderNotRanked         = "leveling_provider_not_ranked"
	ReasonLevelingSignalUnknown             = "leveling_signal_unknown"
	ReasonLevelingMappingInvalid            = "leveling_mapping_invalid"
	ReasonLevelingBudgetInvalid             = "leveling_budget_invalid"
	ReasonLevelingBudgetUnavailable         = "leveling_budget_unavailable"
	ReasonLevelingBudgetExhausted           = "leveling_budget_exhausted"
	ReasonLevelingBudgetDepthExceeded       = "leveling_budget_depth_exceeded"
	ReasonLevelingBudgetParallelism         = "leveling_budget_parallelism_exceeded"
	ReasonLevelingBudgetPermissionDenied    = "leveling_budget_permission_denied"
	ReasonLevelingBudgetIdentityMismatch    = "leveling_budget_identity_mismatch"
	ReasonLevelingBudgetContextExceeded     = "leveling_budget_context_exceeded"
	ReasonLevelingBudgetEvidenceExceeded    = "leveling_budget_evidence_exceeded"
	ReasonLevelingBudgetEscalationDenied    = "leveling_budget_escalation_denied"
	ReasonLevelingBudgetEscalationExhausted = "leveling_budget_escalation_exhausted"
)

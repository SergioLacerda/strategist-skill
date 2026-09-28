package leveling

// Stable diagnostic reason codes emitted by LEVELING validation and authority
// checks. They are catalogued as machine-enforced because callers return these
// errors on their reachable activation paths.
const (
	ReasonLevelingLedgerRecordOversized    = "leveling_ledger_record_oversized"
	ReasonLevelingLockUnavailable          = "leveling_ledger_lock_unavailable"
	ReasonLevelingPolicyStale              = "leveling_policy_stale"
	ReasonLevelingPolicyDigestMismatch     = "leveling_policy_digest_mismatch"
	ReasonLevelingPolicyMissing            = "leveling_policy_missing"
	ReasonLevelingPolicyUnreadable         = "leveling_policy_unreadable"
	ReasonLevelingRankedProviderIneligible = "leveling_ranked_provider_ineligible"
	ReasonLevelingSignalUnknown            = "leveling_signal_unknown"
	ReasonLevelingMappingInvalid           = "leveling_mapping_invalid"
)

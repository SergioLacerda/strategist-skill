package runbook

import "time"

// SelectionPolicy bounds Select's output, per runbook_v2.txt's explicit
// stance against silent, unreasoned runbook application ("eu não aplicaria
// runbooks automaticamente sem explicar a seleção").
//
// TokenBudget, MaxAge, Now, and MinTrust are all additive (design.md § 3
// "Filter by task type and keywords... Enforce trust and staleness policy...
// Apply source and token budgets", 20260819-strategist-improvements/design.md).
// Each is disabled at its zero value, so a policy built before these fields
// existed (or one that simply omits them) selects exactly as it always did —
// only source-count bounds (MaxPrimary/MaxSupporting) and matching apply.
type SelectionPolicy struct {
	MaxPrimary    int
	MaxSupporting int
	RequireReason bool

	// TokenBudget caps the sum of selected candidates' EstimatedTokens. 0
	// (default) disables the check — candidates with EstimatedTokens=0
	// never count against any budget regardless of this field.
	TokenBudget int
	// MaxAge, when > 0, rejects a candidate whose Metadata.ReviewedAt is
	// empty, unparseable, or older than MaxAge relative to Now — the
	// conservative reading of design.md's "staleness policy": an unreviewed
	// or unparseable date is treated as stale, not as fresh-by-default.
	MaxAge time.Duration
	// Now is the reference time MaxAge is measured against. Zero value
	// falls back to time.Now().UTC() at call time — tests that exercise
	// MaxAge should set Now explicitly for determinism.
	Now time.Time
	// MinTrust, when set to one of T0/T1/T2/T3, rejects a candidate whose
	// Trust is a lower tier (T1 is lower than T0, etc.). A candidate with an
	// empty or unrecognized Trust is never rejected by this check — trust
	// is chest-level metadata the caller may not always have propagated,
	// and failing open on an absent signal is safer than silently dropping
	// a candidate over missing bookkeeping.
	MinTrust string
}

// DefaultSelectionPolicy returns runbook_v2.txt's own defaults: at most one
// primary runbook, at most two supporting, and a reason always required.
// TokenBudget, MaxAge, and MinTrust are left at their disabled zero values —
// callers that need those checks set them explicitly.
func DefaultSelectionPolicy() SelectionPolicy {
	return SelectionPolicy{MaxPrimary: 1, MaxSupporting: 2, RequireReason: true}
}

// Rejection records why a candidate Select considered was not selected —
// the auditability half of design.md item 4 ("Record selection and
// rejection reasons"). Every non-selected candidate produces exactly one
// Rejection; RejectionReason values are stable strings suitable for
// logging or telemetry, not free text.
type Rejection struct {
	RunbookID string
	Reason    string
}

// RejectionReason values Rejection.Reason takes.
const (
	// RejectionNoMatch means no applies_when entry matched any signal —
	// the original, pre-existing "never select a zero-match candidate" rule.
	RejectionNoMatch = "no_matching_signal"
	// RejectionStale means policy.MaxAge is set and the candidate's
	// Metadata.ReviewedAt is empty, unparseable, or older than MaxAge.
	RejectionStale = "stale"
	// RejectionConflictsHigherTrust means the candidate's
	// ConflictsWithHigherTrust flag is set — the caller has already
	// determined this runbook contradicts a more authoritative (e.g. T0)
	// source, mirroring context-enrichment.yaml's chest_stop_conditions
	// source_stale_conflicts_T0: prefer_T0 rule.
	RejectionConflictsHigherTrust = "conflicts_with_higher_trust_source"
	// RejectionBelowMinTrust means policy.MinTrust is set and the
	// candidate's Trust is a lower tier.
	RejectionBelowMinTrust = "below_minimum_trust"
	// RejectionOverBudget means policy.TokenBudget is set and adding this
	// candidate's EstimatedTokens would exceed the remaining budget.
	// Select keeps evaluating lower-ranked candidates after an over-budget
	// rejection rather than stopping outright — a smaller candidate later
	// in the ranking may still fit.
	RejectionOverBudget = "over_budget"
	// RejectionPolicyCapReached means the candidate matched and was
	// otherwise eligible, but MaxPrimary and MaxSupporting were both
	// already filled by higher-ranked candidates.
	RejectionPolicyCapReached = "policy_cap_reached"
)

// trustRank orders the closed T0-T3 trust vocabulary (source-card.schema.yaml,
// treasure-chests.yaml#chests[].trust.tier) from most (0) to least (3) trusted.
var trustRank = map[string]int{"T0": 0, "T1": 1, "T2": 2, "T3": 3}

// meetsMinTrust reports whether candidateTrust satisfies minTrust. Either
// side being empty or unrecognized fails open (returns true) — see
// SelectionPolicy.MinTrust's doc comment for why.
func meetsMinTrust(candidateTrust, minTrust string) bool {
	if minTrust == "" || candidateTrust == "" {
		return true
	}
	cr, ok1 := trustRank[candidateTrust]
	mr, ok2 := trustRank[minTrust]
	if !ok1 || !ok2 {
		return true
	}
	return cr <= mr
}

// reviewedAt parses rb.Metadata.ReviewedAt (the "2026-08-20" date format
// used throughout docs/runbooks/*.runbook.yaml sidecars). ok is false when
// the field is empty or fails to parse.
func (rb Runbook) reviewedAt() (t time.Time, ok bool) {
	if rb.Metadata.ReviewedAt == "" {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", rb.Metadata.ReviewedAt)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// isStale reports whether rb fails policy's MaxAge check. Always false when
// MaxAge is 0 (disabled).
func isStale(rb Runbook, policy SelectionPolicy) bool {
	if policy.MaxAge <= 0 {
		return false
	}
	reviewedAt, ok := rb.reviewedAt()
	if !ok {
		return true
	}
	now := policy.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return now.Sub(reviewedAt) > policy.MaxAge
}

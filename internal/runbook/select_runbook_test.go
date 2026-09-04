package runbook

import (
	"testing"
	"time"
)

func candidateRunbooks() []Runbook {
	return []Runbook{
		{RunbookID: "verifying-test-failures", AppliesWhen: []string{"CI test suite is red", "flaky test suspected"}},
		{RunbookID: "verifying-dependency-upgrades", AppliesWhen: []string{"go.mod dependency bumped", "CI test suite is red"}},
		{RunbookID: "release-tool-version-drift", AppliesWhen: []string{"release tooling version mismatch"}},
	}
}

func TestSelect_PicksHighestMatchAsPrimary(t *testing.T) {
	t.Parallel()
	signals := MissionSignals{"CI test suite is red", "flaky test suspected"}
	selections, _, err := Select(candidateRunbooks(), signals, DefaultSelectionPolicy())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selections) == 0 {
		t.Fatal("expected at least one selection")
	}
	if selections[0].RunbookID != "verifying-test-failures" {
		t.Fatalf("expected verifying-test-failures as top match, got %q", selections[0].RunbookID)
	}
	if selections[0].Role != RolePrimary {
		t.Fatalf("expected top match to be primary, got %q", selections[0].Role)
	}
}

func TestSelect_NeverSelectsZeroMatchCandidate(t *testing.T) {
	t.Parallel()
	signals := MissionSignals{"CI test suite is red"}
	selections, rejections, err := Select(candidateRunbooks(), signals, DefaultSelectionPolicy())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, s := range selections {
		if s.RunbookID == "release-tool-version-drift" {
			t.Fatal("release-tool-version-drift has no matching applies_when entry and must never be selected")
		}
	}
	if !hasRejection(rejections, "release-tool-version-drift", RejectionNoMatch) {
		t.Fatalf("expected release-tool-version-drift rejected with reason %q, got %v", RejectionNoMatch, rejections)
	}
}

func TestSelect_NeverReturnsEmptyReasonWhenRequireReasonTrue(t *testing.T) {
	t.Parallel()
	policy := SelectionPolicy{MaxPrimary: 1, MaxSupporting: 5, RequireReason: true}
	selections, _, err := Select(candidateRunbooks(), MissionSignals{"CI test suite is red"}, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selections) == 0 {
		t.Fatal("expected at least one selection to check")
	}
	for _, s := range selections {
		if s.Reason == "" {
			t.Errorf("selection %q has empty reason despite RequireReason=true", s.RunbookID)
		}
	}
}

func TestSelect_RespectsMaxSupporting(t *testing.T) {
	t.Parallel()
	policy := SelectionPolicy{MaxPrimary: 1, MaxSupporting: 1, RequireReason: true}
	selections, _, err := Select(candidateRunbooks(), MissionSignals{"CI test suite is red"}, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	supporting := 0
	for _, s := range selections {
		if s.Role == RoleSupporting {
			supporting++
		}
	}
	if supporting > 1 {
		t.Fatalf("expected at most 1 supporting selection, got %d", supporting)
	}
}

func TestSelect_MatchedButCappedCandidateRejectedAsPolicyCapReached(t *testing.T) {
	t.Parallel()
	// verifying-test-failures outscores verifying-dependency-upgrades on
	// signal {"CI test suite is red"} (its "flaky test suspected" trigger
	// also matches that signal via the canonical vocabulary — see
	// select_runbook_golden_test.go), so with MaxSupporting=0 it alone
	// takes the sole primary slot and verifying-dependency-upgrades — a
	// genuine match, just outranked — is bounded out.
	policy := SelectionPolicy{MaxPrimary: 1, MaxSupporting: 0, RequireReason: true}
	selections, rejections, err := Select(candidateRunbooks(), MissionSignals{"CI test suite is red"}, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selections) != 1 || selections[0].RunbookID != "verifying-test-failures" {
		t.Fatalf("expected only verifying-test-failures selected, got %v", selections)
	}
	if !hasRejection(rejections, "verifying-dependency-upgrades", RejectionPolicyCapReached) {
		t.Fatalf("expected verifying-dependency-upgrades rejected as %q, got %v", RejectionPolicyCapReached, rejections)
	}
}

func TestSelect_RejectsNegativePolicy(t *testing.T) {
	t.Parallel()
	_, _, err := Select(candidateRunbooks(), MissionSignals{"CI test suite is red"}, SelectionPolicy{MaxPrimary: -1})
	if err == nil {
		t.Fatal("expected error for negative max_primary")
	}
}

func TestSelect_RejectsDuplicateCandidateIDs(t *testing.T) {
	t.Parallel()
	dup := []Runbook{
		{RunbookID: "dup", AppliesWhen: []string{"x"}},
		{RunbookID: "dup", AppliesWhen: []string{"y"}},
	}
	_, _, err := Select(dup, MissionSignals{"x"}, DefaultSelectionPolicy())
	if err == nil {
		t.Fatal("expected error for duplicate runbook_id")
	}
}

func TestSelect_NoMatchesReturnsEmptyNotError(t *testing.T) {
	t.Parallel()
	selections, _, err := Select(candidateRunbooks(), MissionSignals{"totally unrelated signal"}, DefaultSelectionPolicy())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selections) != 0 {
		t.Fatalf("expected no selections, got %v", selections)
	}
}

// --- staleness (SelectionPolicy.MaxAge) ---

func TestSelect_StaleCandidateRejectedAndNeverCompetesForASlot(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	candidates := []Runbook{
		{RunbookID: "stale-runbook", AppliesWhen: []string{"CI test suite is red"}, Metadata: Metadata{ReviewedAt: "2020-01-01"}},
	}
	policy := SelectionPolicy{MaxPrimary: 1, MaxSupporting: 2, RequireReason: true, MaxAge: 30 * 24 * time.Hour, Now: now}
	selections, rejections, err := Select(candidates, MissionSignals{"CI test suite is red"}, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selections) != 0 {
		t.Fatalf("expected stale candidate to never be selected, got %v", selections)
	}
	if !hasRejection(rejections, "stale-runbook", RejectionStale) {
		t.Fatalf("expected stale-runbook rejected as %q, got %v", RejectionStale, rejections)
	}
}

func TestSelect_MissingReviewedAtTreatedAsStaleWhenMaxAgeSet(t *testing.T) {
	t.Parallel()
	candidates := []Runbook{
		{RunbookID: "no-reviewed-at", AppliesWhen: []string{"CI test suite is red"}},
	}
	policy := SelectionPolicy{MaxPrimary: 1, RequireReason: true, MaxAge: 24 * time.Hour, Now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	selections, rejections, err := Select(candidates, MissionSignals{"CI test suite is red"}, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selections) != 0 {
		t.Fatalf("expected candidate with no reviewed_at to be rejected as stale, got %v", selections)
	}
	if !hasRejection(rejections, "no-reviewed-at", RejectionStale) {
		t.Fatalf("expected no-reviewed-at rejected as %q, got %v", RejectionStale, rejections)
	}
}

func TestSelect_FreshCandidateNotRejectedByMaxAge(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	candidates := []Runbook{
		{RunbookID: "fresh-runbook", AppliesWhen: []string{"CI test suite is red"}, Metadata: Metadata{ReviewedAt: "2026-08-19"}},
	}
	policy := SelectionPolicy{MaxPrimary: 1, RequireReason: true, MaxAge: 30 * 24 * time.Hour, Now: now}
	selections, _, err := Select(candidates, MissionSignals{"CI test suite is red"}, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selections) != 1 {
		t.Fatalf("expected fresh candidate to be selected, got %v", selections)
	}
}

func TestSelect_MaxAgeZeroDisablesStalenessCheck(t *testing.T) {
	t.Parallel()
	candidates := []Runbook{
		{RunbookID: "no-metadata-at-all", AppliesWhen: []string{"CI test suite is red"}},
	}
	selections, _, err := Select(candidates, MissionSignals{"CI test suite is red"}, DefaultSelectionPolicy())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selections) != 1 {
		t.Fatalf("expected candidate to be selected when MaxAge is disabled, got %v", selections)
	}
}

// --- conflicting (Runbook.ConflictsWithHigherTrust) ---

func TestSelect_ConflictingCandidateRejectedRegardlessOfMatchQuality(t *testing.T) {
	t.Parallel()
	candidates := []Runbook{
		{RunbookID: "conflicting-runbook", AppliesWhen: []string{"CI test suite is red"}, ConflictsWithHigherTrust: true},
	}
	selections, rejections, err := Select(candidates, MissionSignals{"CI test suite is red"}, DefaultSelectionPolicy())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selections) != 0 {
		t.Fatalf("expected conflicting candidate never selected, got %v", selections)
	}
	if !hasRejection(rejections, "conflicting-runbook", RejectionConflictsHigherTrust) {
		t.Fatalf("expected conflicting-runbook rejected as %q, got %v", RejectionConflictsHigherTrust, rejections)
	}
}

// --- trust (SelectionPolicy.MinTrust) ---

func TestSelect_BelowMinTrustCandidateRejected(t *testing.T) {
	t.Parallel()
	candidates := []Runbook{
		{RunbookID: "low-trust", AppliesWhen: []string{"CI test suite is red"}, Trust: "T3"},
	}
	policy := SelectionPolicy{MaxPrimary: 1, RequireReason: true, MinTrust: "T1"}
	selections, rejections, err := Select(candidates, MissionSignals{"CI test suite is red"}, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selections) != 0 {
		t.Fatalf("expected T3 candidate rejected under MinTrust=T1, got %v", selections)
	}
	if !hasRejection(rejections, "low-trust", RejectionBelowMinTrust) {
		t.Fatalf("expected low-trust rejected as %q, got %v", RejectionBelowMinTrust, rejections)
	}
}

func TestSelect_UnknownTrustFailsOpenUnderMinTrust(t *testing.T) {
	t.Parallel()
	candidates := []Runbook{
		{RunbookID: "unknown-trust", AppliesWhen: []string{"CI test suite is red"}},
	}
	policy := SelectionPolicy{MaxPrimary: 1, RequireReason: true, MinTrust: "T0"}
	selections, _, err := Select(candidates, MissionSignals{"CI test suite is red"}, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selections) != 1 {
		t.Fatalf("expected candidate with unset Trust to fail open under MinTrust, got %v", selections)
	}
}

// --- token budget (SelectionPolicy.TokenBudget) ---

func TestSelect_OverBudgetCandidateRejectedButSmallerOneStillFits(t *testing.T) {
	t.Parallel()
	candidates := []Runbook{
		{RunbookID: "big-runbook", AppliesWhen: []string{"CI test suite is red", "flaky test suspected"}, EstimatedTokens: 900},
		{RunbookID: "small-runbook", AppliesWhen: []string{"CI test suite is red"}, EstimatedTokens: 50},
	}
	policy := SelectionPolicy{MaxPrimary: 2, MaxSupporting: 2, RequireReason: true, TokenBudget: 100}
	selections, rejections, err := Select(candidates, MissionSignals{"CI test suite is red", "flaky test suspected"}, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selections) != 1 || selections[0].RunbookID != "small-runbook" {
		t.Fatalf("expected only small-runbook selected within budget, got %v", selections)
	}
	if !hasRejection(rejections, "big-runbook", RejectionOverBudget) {
		t.Fatalf("expected big-runbook rejected as %q, got %v", RejectionOverBudget, rejections)
	}
}

func TestSelect_TokenBudgetZeroDisablesBudgetCheck(t *testing.T) {
	t.Parallel()
	candidates := []Runbook{
		{RunbookID: "huge-runbook", AppliesWhen: []string{"CI test suite is red"}, EstimatedTokens: 1_000_000},
	}
	selections, _, err := Select(candidates, MissionSignals{"CI test suite is red"}, DefaultSelectionPolicy())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selections) != 1 {
		t.Fatalf("expected candidate selected when TokenBudget is disabled, got %v", selections)
	}
}

func hasRejection(rejections []Rejection, runbookID, reason string) bool {
	for _, r := range rejections {
		if r.RunbookID == runbookID && r.Reason == reason {
			return true
		}
	}
	return false
}

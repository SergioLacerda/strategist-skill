package runbook

import (
	"strings"
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

// A placeholder group such as <discovery|refinement|execution> names the values a
// slot may take; it is not a situation. A generic mission signal that equals one
// of those values must not select the runbook.
func TestSelect_SignalInsideAPlaceholderGroupDoesNotMatch(t *testing.T) {
	t.Parallel()
	candidates := []Runbook{{
		RunbookID:   "provider-fallback-policy",
		AppliesWhen: []string{"error=slot_risk_mismatch observed, with slot=<discovery|refinement|execution> in the error context"},
	}}
	for _, signal := range []string{"refinement", "discovery", "execution", "diagnostic"} {
		assertRejectedAsNoMatch(t, candidates, signal)
	}
}

func assertRejectedAsNoMatch(t *testing.T, candidates []Runbook, signal string) {
	t.Helper()
	selections, rejections, err := Select(candidates, MissionSignals{signal}, DefaultSelectionPolicy())
	if err != nil {
		t.Fatalf("signal %q: unexpected error: %v", signal, err)
	}
	if len(selections) != 0 {
		t.Fatalf("signal %q selected %+v through a placeholder group", signal, selections)
	}
	if len(rejections) != 1 || rejections[0].Reason != "no_matching_signal" {
		t.Fatalf("signal %q: want one no_matching_signal rejection, got %+v", signal, rejections)
	}
}

// The raw fallback matches whole words: a signal that only appears inside a longer
// word is not a match, while the same signal as a word or phrase still is.
func TestSelect_RawSignalMatchesWholeWordsOnly(t *testing.T) {
	t.Parallel()
	candidates := []Runbook{{RunbookID: "verifying-test-failures", AppliesWhen: []string{"flaky test suspected in the pipeline"}}}
	cases := map[string]bool{
		"flaky test":                true,
		"pipeline":                  true,
		"Flaky Test Suspected":      true,
		"pipe":                      false,
		"test suspect":              false,
		"error=x":                   false,
		"suspected in the pipeline": true,
	}
	for signal, want := range cases {
		selections, _, err := Select(candidates, MissionSignals{signal}, DefaultSelectionPolicy())
		if err != nil {
			t.Fatalf("signal %q: unexpected error: %v", signal, err)
		}
		if got := len(selections) == 1; got != want {
			t.Fatalf("signal %q: matched=%v, want %v", signal, got, want)
		}
	}
}

// Signals and triggers written with punctuation (error tokens) keep matching as before.
func TestSelect_RawSignalWithPunctuationStillMatches(t *testing.T) {
	t.Parallel()
	candidates := []Runbook{{RunbookID: "role-invocation-failed", AppliesWhen: []string{"error=role_invocation_failed observed, with slot=<discovery|refinement|execution> in the error context"}}}
	selections, _, err := Select(candidates, MissionSignals{"error=role_invocation_failed"}, DefaultSelectionPolicy())
	if err != nil || len(selections) != 1 {
		t.Fatalf("selections=%+v err=%v", selections, err)
	}
}

// A declared Signals value matches a mission signal through the same
// canonical-vocabulary/alias resolution applies_when prose uses, even when
// AppliesWhen itself shares no substring with the mission signal at all.
func TestSelect_DeclaredSignalMatchesWithNoAppliesWhenOverlap(t *testing.T) {
	t.Parallel()
	candidates := []Runbook{{
		RunbookID:   "deep-analysis-workflow",
		AppliesWhen: []string{"quarterly compliance sweep of an unrelated subsystem"},
		Signals:     []string{string(SignalSkillCorpusHealthReview)},
	}}
	selections, _, err := Select(candidates, MissionSignals{"hardening"}, DefaultSelectionPolicy())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selections) != 1 || selections[0].RunbookID != "deep-analysis-workflow" {
		t.Fatalf("expected declared Signals to select the candidate, got %+v", selections)
	}
	if !strings.Contains(selections[0].Reason, "signal:"+string(SignalSkillCorpusHealthReview)) {
		t.Fatalf("expected reason to name the matched signal, got %q", selections[0].Reason)
	}
}

// Signals supplements AppliesWhen rather than replacing it: a candidate
// whose AppliesWhen matches but whose Signals does not is still selected on
// the strength of the prose match alone.
func TestSelect_UnmatchedDeclaredSignalDoesNotSuppressAppliesWhenMatch(t *testing.T) {
	t.Parallel()
	candidates := []Runbook{{
		RunbookID:   "verifying-test-failures",
		AppliesWhen: []string{"CI test suite is red"},
		Signals:     []string{string(SignalDependencyUpgrade)},
	}}
	selections, _, err := Select(candidates, MissionSignals{"CI test suite is red"}, DefaultSelectionPolicy())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selections) != 1 {
		t.Fatalf("expected the applies_when match to still select the candidate, got %+v", selections)
	}
}

// A candidate with no declared Signals (every sidecar in the tree today)
// behaves exactly as it did before the field existed.
func TestSelect_EmptyDeclaredSignalsIsByteForByteUnchanged(t *testing.T) {
	t.Parallel()
	signals := MissionSignals{"CI test suite is red", "flaky test suspected"}
	selections, _, err := Select(candidateRunbooks(), signals, DefaultSelectionPolicy())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selections) == 0 || selections[0].RunbookID != "verifying-test-failures" {
		t.Fatalf("expected unchanged behavior for candidates without declared Signals, got %+v", selections)
	}
}

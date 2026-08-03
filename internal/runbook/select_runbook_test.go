package runbook

import "testing"

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
	selections, err := Select(candidateRunbooks(), signals, DefaultSelectionPolicy())
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
	selections, err := Select(candidateRunbooks(), signals, DefaultSelectionPolicy())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, s := range selections {
		if s.RunbookID == "release-tool-version-drift" {
			t.Fatal("release-tool-version-drift has no matching applies_when entry and must never be selected")
		}
	}
}

func TestSelect_NeverReturnsEmptyReasonWhenRequireReasonTrue(t *testing.T) {
	t.Parallel()
	policy := SelectionPolicy{MaxPrimary: 1, MaxSupporting: 5, RequireReason: true}
	selections, err := Select(candidateRunbooks(), MissionSignals{"CI test suite is red"}, policy)
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
	selections, err := Select(candidateRunbooks(), MissionSignals{"CI test suite is red"}, policy)
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

func TestSelect_RejectsNegativePolicy(t *testing.T) {
	t.Parallel()
	_, err := Select(candidateRunbooks(), MissionSignals{"CI test suite is red"}, SelectionPolicy{MaxPrimary: -1})
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
	_, err := Select(dup, MissionSignals{"x"}, DefaultSelectionPolicy())
	if err == nil {
		t.Fatal("expected error for duplicate runbook_id")
	}
}

func TestSelect_NoMatchesReturnsEmptyNotError(t *testing.T) {
	t.Parallel()
	selections, err := Select(candidateRunbooks(), MissionSignals{"totally unrelated signal"}, DefaultSelectionPolicy())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selections) != 0 {
		t.Fatalf("expected no selections, got %v", selections)
	}
}

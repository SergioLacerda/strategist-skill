package telemetry

import "testing"

func TestComputeHandoffMetrics_EmptyInput(t *testing.T) {
	t.Parallel()
	m := ComputeHandoffMetrics(nil)
	if m.HandoffPassRate != 0 || m.FirstAttemptPassRate != 0 || m.CriticalConstraintRecall != 0 ||
		m.DecisionClassificationAccuracy != 0 || m.ScopeViolationRate != 0 || m.HandoffRepairRate != 0 {
		t.Fatalf("expected all rates 0 for empty input, got %+v", m)
	}
	if m.SemanticLoss != (SemanticHandoffLoss{}) {
		t.Fatalf("expected zero semantic loss, got %+v", m.SemanticLoss)
	}
	if m.SampleSize != 0 {
		t.Fatalf("expected zero sample size, got %d", m.SampleSize)
	}
}

func TestComputeHandoffMetrics_HandoffPassRate(t *testing.T) {
	t.Parallel()
	records := []ChallengeRecord{
		{MissionID: "m-1", Attempt: 1, Passed: true},
		{MissionID: "m-2", Attempt: 1, Passed: false},
		{MissionID: "m-3", Attempt: 1, Passed: true},
		{MissionID: "m-4", Attempt: 1, Passed: true},
	}
	m := ComputeHandoffMetrics(records)
	if got, want := m.HandoffPassRate, 0.75; got != want {
		t.Fatalf("HandoffPassRate = %v, want %v", got, want)
	}
	if m.SampleSize != 4 {
		t.Fatalf("SampleSize = %d, want 4", m.SampleSize)
	}
}

func TestComputeHandoffMetrics_FirstAttemptPassRateIgnoresLaterAttempts(t *testing.T) {
	t.Parallel()
	records := []ChallengeRecord{
		{MissionID: "m-1", Attempt: 1, Passed: false},
		{MissionID: "m-1", Attempt: 2, Passed: true}, // repaired, must not count toward first-attempt rate
		{MissionID: "m-2", Attempt: 1, Passed: true},
	}
	m := ComputeHandoffMetrics(records)
	if got, want := m.FirstAttemptPassRate, 0.5; got != want {
		t.Fatalf("FirstAttemptPassRate = %v, want %v", got, want)
	}
}

func TestComputeHandoffMetrics_CriticalConstraintRecallAndSemanticLoss(t *testing.T) {
	t.Parallel()
	records := []ChallengeRecord{
		{MissionID: "m-1", Attempt: 1, MissingRefs: nil},
		{MissionID: "m-2", Attempt: 1, MissingRefs: []string{"X-001"}},
	}
	m := ComputeHandoffMetrics(records)
	if got, want := m.CriticalConstraintRecall, 0.5; got != want {
		t.Fatalf("CriticalConstraintRecall = %v, want %v", got, want)
	}
	if got, want := m.SemanticLoss.Recall, 0.5; got != want {
		t.Fatalf("SemanticLoss.Recall = %v, want %v", got, want)
	}
	if m.SemanticLoss.Application != 0 {
		t.Fatalf("expected SemanticLoss.Application always 0 (no ground truth signal), got %v", m.SemanticLoss.Application)
	}
}

func TestComputeHandoffMetrics_DecisionClassificationAccuracy(t *testing.T) {
	t.Parallel()
	records := []ChallengeRecord{
		{MissionID: "m-1", Attempt: 1, MisclassifiedRefs: nil},
		{MissionID: "m-2", Attempt: 1, MisclassifiedRefs: nil},
		{MissionID: "m-3", Attempt: 1, MisclassifiedRefs: []string{"VERDICT"}},
	}
	m := ComputeHandoffMetrics(records)
	accuracy := float64(2) / float64(3) // matches safeRate's runtime float64 division exactly
	if got, want := m.DecisionClassificationAccuracy, accuracy; got != want {
		t.Fatalf("DecisionClassificationAccuracy = %v, want %v", got, want)
	}
	if got, want := m.SemanticLoss.Classification, 1-accuracy; got != want {
		t.Fatalf("SemanticLoss.Classification = %v, want %v", got, want)
	}
}

func TestComputeHandoffMetrics_ScopeViolationRate(t *testing.T) {
	t.Parallel()
	records := []ChallengeRecord{
		{MissionID: "m-1", Attempt: 1, MissingChallenges: []string{"boundary"}},
		{MissionID: "m-2", Attempt: 1, MissingChallenges: []string{"recall"}},
	}
	m := ComputeHandoffMetrics(records)
	if got, want := m.ScopeViolationRate, 0.5; got != want {
		t.Fatalf("ScopeViolationRate = %v, want %v", got, want)
	}
}

func TestComputeHandoffMetrics_HandoffRepairRate(t *testing.T) {
	t.Parallel()
	records := []ChallengeRecord{
		// m-1: failed first attempt, repaired on attempt 2.
		{MissionID: "m-1", Attempt: 1, Passed: false},
		{MissionID: "m-1", Attempt: 2, Passed: true},
		// m-2: failed first attempt, no later attempt recorded (not repaired).
		{MissionID: "m-2", Attempt: 1, Passed: false},
		// m-3: passed first attempt, never needed repair — excluded from denominator.
		{MissionID: "m-3", Attempt: 1, Passed: true},
	}
	m := ComputeHandoffMetrics(records)
	if got, want := m.HandoffRepairRate, 0.5; got != want {
		t.Fatalf("HandoffRepairRate = %v, want %v", got, want)
	}
}

func TestComputeHandoffMetrics_HandoffRepairRateZeroWhenNoFailures(t *testing.T) {
	t.Parallel()
	records := []ChallengeRecord{
		{MissionID: "m-1", Attempt: 1, Passed: true},
	}
	m := ComputeHandoffMetrics(records)
	if m.HandoffRepairRate != 0 {
		t.Fatalf("expected 0 HandoffRepairRate with no first-attempt failures, got %v", m.HandoffRepairRate)
	}
}

func TestComputeHandoffMetrics_HandoffRepairRateIgnoresMissionWithNoFirstAttempt(t *testing.T) {
	t.Parallel()
	records := []ChallengeRecord{
		// m-1: no attempt 1 on record at all (e.g. history started being
		// captured mid-mission) — must be excluded from the repair-rate
		// denominator entirely, not treated as a first-attempt failure.
		{MissionID: "m-1", Attempt: 2, Passed: false},
		{MissionID: "m-1", Attempt: 3, Passed: true},
		// m-2: failed first attempt, no later attempt — counts as
		// failedFirst but not repaired.
		{MissionID: "m-2", Attempt: 1, Passed: false},
	}
	m := ComputeHandoffMetrics(records)
	if got, want := m.HandoffRepairRate, 0.0; got != want {
		t.Fatalf("HandoffRepairRate = %v, want %v (m-1 must not count toward the denominator)", got, want)
	}
}

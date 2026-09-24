package initiative

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultPolicyIsDeterministicAndValidated(t *testing.T) {
	policy := DefaultPolicy()
	require.NoError(t, policy.Validate(), "default policy must validate")

	require.NotEmpty(t, policy.Digest(), "policy digest must be stable")
	require.Equal(t, policy.Digest(), DefaultPolicy().Digest(), "policy digest must be stable")

	changed := DefaultPolicy()
	changed.Profiles["ranger"] = Profile{RecommendedCapability: "economical", RecommendedEffort: EffortLow, Diligence: []string{"one"}, ConfidenceCeiling: "low"}
	require.NotEqual(t, policy.Digest(), changed.Digest(), "policy digest must identify changed policy")

	invalid := Policy{Version: "1", Profiles: map[string]Profile{"ranger": {RecommendedCapability: "reasoning", RecommendedEffort: "invalid", Diligence: []string{"x"}, ConfidenceCeiling: "low"}}}
	err := invalid.Validate()
	require.Error(t, err, "expected invalid effort error")
	require.Contains(t, err.Error(), "invalid recommended effort")

	parsed, err := Parse([]byte("version: '1'\nprofiles:\n  ranger:\n    recommended_capability: reasoning\n    recommended_effort: high\n    diligence: [inspect]\n    confidence_ceiling: high\nreevaluation_triggers: [scope_changed]\n"))
	require.NoError(t, err, "standalone policy parse failed")
	require.NotEmpty(t, parsed.Digest(), "standalone policy parse failed")
}

func TestAdvisorKeepsLevelingAuthoritySeparate(t *testing.T) {
	policy := DefaultPolicy()
	advisor := Advisor{Policy: policy}
	advice, err := advisor.Advise(AdviceInput{
		MissionID: "m-1", Role: "Ranger", RunID: "run-1", Trigger: TriggerInitial,
		Observed: Observation{State: ObservationKnown, Model: "codex-reasoning", Provider: "CODEX", Effort: EffortMedium, Capability: "reasoning", LevelSource: "host"},
	})
	require.NoError(t, err)

	require.Equal(t, "ranger", advice.Role)
	require.Equal(t, EffortHigh, advice.Recommendation.RecommendedEffort)
	require.Equal(t, AlignmentBelowRecommendation, advice.Alignment)

	require.NotEmpty(t, advice.AdviceID, "advice identity missing")
	require.Equal(t, policy.Digest(), advice.PolicyDigest, "advice identity missing")

	err = ValidateAdviceJSON([]byte(`{"advice_id":"x","model":"should-not-be-here"}`))
	require.Error(t, err, "initiative must reject top-level LEVELING fields")
	require.Contains(t, err.Error(), "authority_violation")

	require.NoError(t, ValidateAdviceJSON(mustJSON(t, advice)), "valid advice JSON rejected")
}

func TestAdvisorCopiesLevelingResolutionBeforeConsultation(t *testing.T) {
	resolution := &LevelingResolution{
		EventID: "level-1", Role: "ranger", State: ObservationKnown,
		Model: "host-model", Provider: "host", Effort: EffortHigh,
		Capability: "reasoning", LevelSource: "host",
	}
	advice, err := (Advisor{Policy: DefaultPolicy()}).Advise(AdviceInput{
		MissionID: "m-level", Role: "ranger", RunID: "run", Trigger: TriggerInitial,
		Leveling: resolution,
	})
	require.NoError(t, err)
	resolution.Model = "mutated-after-consultation"
	resolution.Provider = "mutated-provider"
	require.NotNil(t, advice.Leveling)
	require.Equal(t, "host-model", advice.Leveling.Model)
	require.Equal(t, "host", advice.Leveling.Provider)
	require.Equal(t, "host-model", advice.Observed.Model)
}

func TestAdvisorRejectsLevelingResolutionForAnotherRole(t *testing.T) {
	_, err := (Advisor{Policy: DefaultPolicy()}).Advise(AdviceInput{
		MissionID: "m-level-role", Role: "ranger", RunID: "run", Trigger: TriggerInitial,
		Leveling: &LevelingResolution{EventID: "level-1", Role: "sniper", State: ObservationKnown},
	})
	require.ErrorContains(t, err, "resolution role does not match")
}

func TestRuntimeReevaluationRejectsTheSameLevelingEvent(t *testing.T) {
	runtime, err := NewRuntime(t.TempDir(), DefaultPolicy())
	require.NoError(t, err)
	resolution := &LevelingResolution{EventID: "level-1", Role: "ranger", State: ObservationKnown, Effort: EffortHigh}
	_, _, err = runtime.EnterRole(AdviceInput{MissionID: "m-level-retry", Role: "ranger", RunID: "run", Trigger: TriggerInitial, Leveling: resolution})
	require.NoError(t, err)
	_, err = runtime.Reevaluate(AdviceInput{MissionID: "m-level-retry", Role: "ranger", RunID: "run", Trigger: TriggerScopeChanged, Leveling: resolution})
	require.ErrorContains(t, err, "new LEVELING event")
}

func TestAdvisorReevaluationRequiresSupersession(t *testing.T) {
	advisor := Advisor{Policy: DefaultPolicy()}
	_, err := advisor.Advise(AdviceInput{MissionID: "m", Role: "ranger", RunID: "r", Trigger: TriggerScopeChanged})
	if err == nil || !strings.Contains(err.Error(), "requires supersedes") {
		t.Fatalf("expected supersession requirement, got %v", err)
	}
	first, err := advisor.Advise(AdviceInput{MissionID: "m", Role: "ranger", RunID: "r", Trigger: TriggerInitial})
	if err != nil {
		t.Fatal(err)
	}
	second, err := advisor.Advise(AdviceInput{MissionID: "m", Role: "ranger", RunID: "r", Trigger: TriggerScopeChanged, Supersedes: first.AdviceID, Sequence: 2})
	if err != nil || second.Supersedes != first.AdviceID || second.AdviceID == first.AdviceID {
		t.Fatalf("unexpected reevaluation: advice=%+v err=%v", second, err)
	}
}

func TestRepeatedLocalActionsReuseAdviceID(t *testing.T) {
	advisor := Advisor{Policy: DefaultPolicy()}
	input := AdviceInput{MissionID: "m", Role: "ranger", RunID: "run", Trigger: TriggerInitial, Sequence: 1}
	first, err := advisor.Advise(input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := advisor.Advise(input)
	if err != nil {
		t.Fatal(err)
	}
	if first.AdviceID != second.AdviceID {
		t.Fatalf("local actions must reuse stable advice_id: %q != %q", first.AdviceID, second.AdviceID)
	}
}

func TestAlignmentPreservesUnavailableAndNotComparable(t *testing.T) {
	if got := AlignmentFor(Observation{State: ObservationUnavailable}, EffortHigh); got != AlignmentUnavailable {
		t.Fatalf("unavailable alignment: %s", got)
	}
	if got := AlignmentFor(Observation{State: ObservationUnknown}, EffortHigh); got != AlignmentUnknown {
		t.Fatalf("unknown alignment: %s", got)
	}
	if got := AlignmentFor(Observation{State: ObservationKnown, Effort: "legacy"}, EffortHigh); got != AlignmentNotComparable {
		t.Fatalf("not comparable alignment: %s", got)
	}
}

func TestResultValidationChallengesMissingEvidenceAndBlockedObligation(t *testing.T) {
	advisor := Advisor{Policy: DefaultPolicy()}
	advice, err := advisor.Advise(AdviceInput{MissionID: "m", Role: "archivist", RunID: "r", Trigger: TriggerInitial})
	if err != nil {
		t.Fatal(err)
	}
	result := Result{AdviceID: advice.AdviceID, MissionID: "m", Role: "archivist", RunID: "r", GateIndependent: true, Checks: []ObligationCheck{{ID: "evidence", Status: CheckBlocked}}, EvidenceRefs: nil}
	assessment, err := AssessResult(advice, result)
	if err != nil || !assessment.Challenge || assessment.ConfidenceCeiling != "low" || len(assessment.Reasons) != 2 {
		t.Fatalf("unexpected assessment: %+v err=%v", assessment, err)
	}
	bad := result
	bad.GateIndependent = false
	if _, err := AssessResult(advice, bad); err == nil {
		t.Fatal("result must not authorize the Approval Gate")
	}
	good := Result{AdviceID: advice.AdviceID, MissionID: "m", Role: "archivist", RunID: "r", GateIndependent: true, Checks: []ObligationCheck{{ID: "evidence", Status: CheckSatisfied, EvidenceRefs: []EvidenceRef{{ID: "e-1", Class: "explicit"}}}}, EvidenceRefs: []EvidenceRef{{ID: "e-1", Class: "explicit"}}, Outcomes: []OutcomeCorrelation{{ID: "out-1", Status: "observed", EvidenceRefs: []EvidenceRef{{ID: "e-1", Class: "explicit"}}}}}
	assessment, err = AssessResult(advice, good)
	if err != nil || assessment.Challenge {
		t.Fatalf("evidenced result should remain advisory without challenge: %+v %v", assessment, err)
	}
}

func TestLedgerIsAppendOnlyAndIndependentFromLeveling(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory", "initiative-records.jsonl")
	advisor := Advisor{Policy: DefaultPolicy()}
	advice, err := advisor.Advise(AdviceInput{MissionID: "m", Role: "ranger", RunID: "run", Trigger: TriggerInitial})
	if err != nil {
		t.Fatal(err)
	}
	record := Record{Kind: RecordKindAdvice, MissionID: "m", Role: "ranger", RunID: "run", AdviceID: advice.AdviceID, Advice: &advice, Timestamp: "2026-09-23T12:00:00Z"}
	if err := AppendRecord(path, record); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(mustRead(t, path), []byte("malformed\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	latest, found, err := LatestAdvice(path, "m", "RANGER", "run")
	if err != nil || !found || latest.AdviceID != advice.AdviceID {
		t.Fatalf("latest advice: %+v found=%v err=%v", latest, found, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

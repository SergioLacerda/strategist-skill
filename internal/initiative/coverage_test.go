package initiative

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func validAdvice() Advice {
	return Advice{
		AdviceID:      "adv-1",
		MissionID:     "m",
		Role:          "ranger",
		RunID:         "r",
		PolicyVersion: "1",
		PolicyDigest:  "digest",
		Trigger:       TriggerInitial,
		Recommendation: Recommendation{
			RecommendedCapability: "reasoning",
			RecommendedEffort:     EffortHigh,
		},
		Alignment: AlignmentMatched,
		Diligence: DiligenceProfile{Checks: []string{"inspect"}, ConfidenceCeiling: "high"},
	}
}

func TestAdviceValidateRejectsEachIncompleteField(t *testing.T) {
	require.NoError(t, validAdvice().Validate(), "baseline advice must validate")

	missingIdentity := validAdvice()
	missingIdentity.AdviceID = ""
	require.ErrorContains(t, missingIdentity.Validate(), "identity fields are required")

	unknownTrigger := validAdvice()
	unknownTrigger.Trigger = "bogus"
	require.ErrorContains(t, unknownTrigger.Validate(), "invalid trigger/supersedes pair")

	missingRecommendation := validAdvice()
	missingRecommendation.Recommendation.RecommendedCapability = ""
	require.ErrorContains(t, missingRecommendation.Validate(), "recommendation is incomplete")

	missingDiligence := validAdvice()
	missingDiligence.Diligence.Checks = nil
	require.ErrorContains(t, missingDiligence.Validate(), "diligence or alignment is incomplete")

	invalidAlignment := validAdvice()
	invalidAlignment.Alignment = "bogus"
	require.ErrorContains(t, invalidAlignment.Validate(), "diligence or alignment is incomplete")
}

func TestValidateAdviceJSONDecodeErrors(t *testing.T) {
	err := ValidateAdviceJSON([]byte("not json"))
	require.ErrorContains(t, err, "decode:")

	err = ValidateAdviceJSON([]byte(`{"advice_id": 123}`))
	require.ErrorContains(t, err, "decode envelope")
}

func TestAlignmentForCoversAllRanks(t *testing.T) {
	require.Equal(t, AlignmentAboveRecommendation, AlignmentFor(Observation{State: ObservationKnown, Effort: EffortXHigh}, EffortLow))
	require.Equal(t, AlignmentMatched, AlignmentFor(Observation{State: ObservationKnown, Effort: EffortMedium}, EffortMedium))
}

func TestNormalizeObservationDefaultsStateFromFields(t *testing.T) {
	got := normalizeObservation(Observation{Effort: EffortHigh})
	require.Equal(t, ObservationKnown, got.State)
}

func TestAdviseRejectsInvalidPolicy(t *testing.T) {
	_, err := (Advisor{Policy: Policy{}}).Advise(AdviceInput{MissionID: "m", Role: "ranger", RunID: "r", Trigger: TriggerInitial})
	require.ErrorContains(t, err, "version is required")
}

func TestAdviseRejectsUnknownRole(t *testing.T) {
	_, err := (Advisor{Policy: DefaultPolicy()}).Advise(AdviceInput{MissionID: "m", Role: "unknown-role", RunID: "r", Trigger: TriggerInitial})
	require.ErrorContains(t, err, "is not in policy")
}

func TestAdviseRejectsNegativeSequence(t *testing.T) {
	_, err := (Advisor{Policy: DefaultPolicy()}).Advise(AdviceInput{MissionID: "m", Role: "ranger", RunID: "r", Trigger: TriggerInitial, Sequence: -1})
	require.ErrorContains(t, err, "sequence cannot be negative")
}

func TestAdviseRejectsMissingIdentityFields(t *testing.T) {
	_, err := (Advisor{Policy: DefaultPolicy()}).Advise(AdviceInput{MissionID: "", Role: "ranger", RunID: "r", Trigger: TriggerInitial})
	require.ErrorContains(t, err, "mission_id, role, and run_id are required")
}

func TestAdviseRejectsUnknownTrigger(t *testing.T) {
	_, err := (Advisor{Policy: DefaultPolicy()}).Advise(AdviceInput{MissionID: "m", Role: "ranger", RunID: "r", Trigger: "bogus"})
	require.ErrorContains(t, err, "unknown trigger")
}

func TestPolicyValidateRejectsEmptyProfiles(t *testing.T) {
	err := (Policy{Version: "1"}).Validate()
	require.ErrorContains(t, err, "profiles are required")
}

func TestValidateProfileRejectsMissingCapability(t *testing.T) {
	policy := Policy{Version: "1", Profiles: map[string]Profile{
		"ranger": {RecommendedCapability: "", RecommendedEffort: EffortHigh, Diligence: []string{"x"}, ConfidenceCeiling: "high"},
	}}
	require.ErrorContains(t, policy.Validate(), "has no recommended capability")
}

func TestValidateProfileRejectsMissingDiligence(t *testing.T) {
	policy := Policy{Version: "1", Profiles: map[string]Profile{
		"ranger": {RecommendedCapability: "reasoning", RecommendedEffort: EffortHigh, Diligence: nil, ConfidenceCeiling: "high"},
	}}
	require.ErrorContains(t, policy.Validate(), "requires diligence and confidence ceiling")
}

func TestValidateReevaluationTriggersRejectsInvalid(t *testing.T) {
	policy := DefaultPolicy()
	policy.Triggers = append(policy.Triggers, TriggerInitial)
	require.ErrorContains(t, policy.Validate(), "invalid re-evaluation trigger")
}

func TestPolicyRejectsDuplicateReevaluationTriggers(t *testing.T) {
	policy := DefaultPolicy()
	policy.Triggers = append(policy.Triggers, TriggerScopeChanged)
	require.ErrorContains(t, policy.Validate(), "duplicate re-evaluation trigger")
}

func TestPolicyAllowsOnlyDeclaredReevaluationTriggers(t *testing.T) {
	policy := DefaultPolicy()
	require.True(t, policy.AllowsTrigger(TriggerInitial))
	require.True(t, policy.AllowsTrigger(TriggerScopeChanged))
	require.False(t, policy.AllowsTrigger(Trigger("not_declared")))
}

func TestParseRejectsMalformedYAML(t *testing.T) {
	_, err := Parse([]byte("version: [1\n"))
	require.ErrorContains(t, err, "parse:")
}

func TestParseRejectsInvalidPolicy(t *testing.T) {
	_, err := Parse([]byte("version: '1'\n"))
	require.ErrorContains(t, err, "profiles are required")
}

func TestAppendRecordRejectsInvalidRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory", "initiative-records.jsonl")
	err := AppendRecord(path, Record{Kind: "bogus"})
	require.ErrorContains(t, err, "unknown kind")
}

func TestAppendRecordDefaultsTimestampForAdviceRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory", "initiative-records.jsonl")
	advice := validAdvice()
	record := Record{
		Kind: RecordKindAdvice, MissionID: advice.MissionID, Role: advice.Role, RunID: advice.RunID, AdviceID: advice.AdviceID,
		Advice: &advice,
	}
	require.NoError(t, AppendRecord(path, record))

	records, err := ReadRecords(path)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.NotEmpty(t, records[0].Timestamp, "timestamp must default when omitted")
}

func TestAppendRecordRejectsUnvalidatedResultRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory", "initiative-records.jsonl")
	record := Record{Kind: RecordKindResult, MissionID: "m", Role: "ranger", RunID: "r", AdviceID: "adv-1", Result: &Result{AdviceID: "adv-1"}}
	require.ErrorContains(t, AppendRecord(path, record), "validated runtime")
}

func TestAppendRecordRejectsUnwritableDirectory(t *testing.T) {
	conflict := filepath.Join(t.TempDir(), "not-a-dir")
	require.NoError(t, os.WriteFile(conflict, []byte("x"), 0o600))
	advice := validAdvice()
	record := Record{Kind: RecordKindAdvice, MissionID: advice.MissionID, Role: advice.Role, RunID: advice.RunID, AdviceID: advice.AdviceID, Advice: &advice}
	err := AppendRecord(filepath.Join(conflict, "sub", "initiative-records.jsonl"), record)
	require.ErrorContains(t, err, "create ledger directory")
}

func TestWriteLedgerLineRejectsMissingDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-dir", "records.jsonl")
	err := writeLedgerLine(path, []byte("{}"))
	require.ErrorContains(t, err, "open ledger")
}

func TestReadRecordsHandlesMissingAndInvalidPaths(t *testing.T) {
	records, err := ReadRecords(filepath.Join(t.TempDir(), "does-not-exist.jsonl"))
	require.NoError(t, err)
	require.Nil(t, records)

	invalidPath := string([]byte{0})
	_, err = ReadRecords(invalidPath)
	require.ErrorContains(t, err, "open ledger")
}

func TestScanRecordsRejectsOversizedLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "records.jsonl")
	oversized := bytes.Repeat([]byte("a"), 1<<20)
	require.NoError(t, os.WriteFile(path, append(oversized, '\n'), 0o600))
	_, err := ReadRecords(path)
	require.ErrorContains(t, err, "read ledger")
}

func TestLatestAdvicePropagatesReadError(t *testing.T) {
	invalidPath := string([]byte{0})
	_, _, err := LatestAdvice(invalidPath, "m", "ranger", "r")
	require.Error(t, err)
}

func TestValidateRecordRejectsMissingIdentity(t *testing.T) {
	err := validateRecord(Record{Kind: RecordKindAdvice})
	require.ErrorContains(t, err, "mission, role, run, and advice identity are required")
}

func TestValidateAdviceRecordRejectsMismatch(t *testing.T) {
	err := validateRecord(Record{
		Kind: RecordKindAdvice, MissionID: "m", Role: "ranger", RunID: "r", AdviceID: "a1",
		Advice: &Advice{AdviceID: "different"},
	})
	require.ErrorContains(t, err, "advice record payload mismatch")
}

func TestValidateResultRecordRejectsMismatch(t *testing.T) {
	err := validateRecord(Record{
		Kind: RecordKindResult, MissionID: "m", Role: "ranger", RunID: "r", AdviceID: "a1",
		Result: &Result{AdviceID: "different"},
	})
	require.ErrorContains(t, err, "result record payload mismatch")
}

func TestValidateAgainstRejectsUncorrelatedResult(t *testing.T) {
	advice := validAdvice()
	result := Result{AdviceID: "different"}
	require.ErrorContains(t, result.ValidateAgainst(advice), "does not correlate with advice")
}

func TestValidateAgainstRejectsEmptyChecks(t *testing.T) {
	advice := validAdvice()
	result := Result{AdviceID: advice.AdviceID, MissionID: advice.MissionID, Role: advice.Role, RunID: advice.RunID, GateIndependent: true}
	require.ErrorContains(t, result.ValidateAgainst(advice), "checks are required")
}

func TestValidateAgainstPropagatesCheckError(t *testing.T) {
	advice := validAdvice()
	result := Result{
		AdviceID: advice.AdviceID, MissionID: advice.MissionID, Role: advice.Role, RunID: advice.RunID,
		GateIndependent: true, Checks: []ObligationCheck{{ID: "", Status: CheckSatisfied}},
	}
	require.ErrorContains(t, result.ValidateAgainst(advice), "check id and valid status are required")
}

func TestValidateCheckRejectsSatisfiedWithoutEvidence(t *testing.T) {
	err := validateCheck(ObligationCheck{ID: "x", Status: CheckSatisfied})
	require.ErrorContains(t, err, "requires evidence")
}

func TestValidateOutcomesRejectsIncompleteOutcome(t *testing.T) {
	err := validateOutcomes([]OutcomeCorrelation{{ID: "", Status: "observed"}})
	require.ErrorContains(t, err, "outcome id and status are required")
}

func TestAssessResultChallengesPartialCheck(t *testing.T) {
	advice := validAdvice()
	result := Result{
		AdviceID: advice.AdviceID, MissionID: advice.MissionID, Role: advice.Role, RunID: advice.RunID,
		GateIndependent: true,
		Checks:          []ObligationCheck{{ID: "check-1", Status: CheckPartial}},
		EvidenceRefs:    []EvidenceRef{{ID: "e-1", Class: "explicit"}},
	}
	assessment, err := AssessResult(advice, result)
	require.NoError(t, err)
	require.True(t, assessment.Challenge)
	require.Contains(t, assessment.Reasons, "partial_obligation:check-1")
}

package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func mustFail(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), substr) {
		t.Fatalf("expected error containing %q, got %v", substr, err)
	}
}

func TestValidateCalibrationStatusMatrix(t *testing.T) {
	t.Parallel()
	mustFail(t, ValidateCalibrationStatus("", -1), "negative")
	mustFail(t, ValidateCalibrationStatus(CalibrationObserved, 0), "requires calibration_status")
	if err := ValidateCalibrationStatus("", 0); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCalibrationStatus(CalibrationNoSample, 0); err != nil {
		t.Fatal(err)
	}
	mustFail(t, ValidateCalibrationStatus(CalibrationNoSample, 2), "no_sample requires")
	mustFail(t, ValidateCalibrationStatus("bogus", 2), "not allowed")
	mustFail(t, ValidateCalibrationStatus(CalibrationCalibrated, 1), "at least")
	if err := ValidateCalibrationStatus(CalibrationCalibrated, CalibrationMinimumSample); err != nil {
		t.Fatal(err)
	}
}

func TestValidateConfidenceCompatibility(t *testing.T) {
	t.Parallel()
	if err := validateConfidenceCompatibility(ConfidenceLow, nil); err != nil {
		t.Fatal(err)
	}
	bad, ok := 101, 90
	mustFail(t, validateConfidenceCompatibility("", &bad), "out of range")
	mustFail(t, validateConfidenceCompatibility(ConfidenceLow, &ok), "conflicts")
	if err := validateConfidenceCompatibility(ConfidenceHigh, &ok); err != nil {
		t.Fatal(err)
	}
}

func TestConfidenceClaimUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var c ConfidenceClaim
	mustFail(t, json.Unmarshal([]byte(`{"id":1}`), &c), "decode")
	mustFail(t, json.Unmarshal([]byte(`[]`), &c), "confidence claim")
	if err := json.Unmarshal([]byte(`{"id":"a"}`), &c); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"id":"a","evidence":["e1"]}`), &c); err != nil || len(c.EvidenceIDs) != 1 {
		t.Fatalf("legacy alias: %v %v", err, c.EvidenceIDs)
	}
	if err := json.Unmarshal([]byte(`{"id":"a","evidence":["e1"],"evidence_ids":["e1"]}`), &c); err != nil {
		t.Fatal(err)
	}
	mustFail(t, json.Unmarshal([]byte(`{"id":"a","evidence":["e1"],"evidence_ids":["e2"]}`), &c), "different values")
	mustFail(t, json.Unmarshal([]byte(`{"id":"a","evidence":"x"}`), &c), "evidence compatibility")
	mustFail(t, json.Unmarshal([]byte(`{"id":"a","evidence":["e1"],"evidence_ids":"x"}`), &c), "decode")
}

func TestValidateConfidenceClaimFieldErrors(t *testing.T) {
	t.Parallel()
	claim := ConfidenceClaim{
		ClaimKind: "weird", ConfidencePercent: 150, CalibrationStatus: CalibrationNoSample, SampleSize: 2,
		GroundTruthRef: "r", EvidenceIDs: []string{"a"}, EvidenceClasses: []string{"x", "y"},
	}
	err := ValidateConfidenceClaim(claim, nil)
	for _, want := range []string{"id is required", "statement is required", "claim_kind", "out of range", "no_sample", "ground_truth_ref requires", "must align"} {
		mustFail(t, err, want)
	}
	claim = ConfidenceClaim{
		ID: "a", Statement: "s", ClaimKind: ClaimKindQuestion, ConfidencePercent: 10, ConfidenceLevel: ConfidenceHigh,
		GroundTruthKind: "nope", GroundTruthOutcome: "nope",
	}
	err = ValidateConfidenceClaim(claim, nil)
	for _, want := range []string{"conflicts with derived", "ground_truth_kind", "ground_truth_outcome", "requires ground_truth_ref"} {
		mustFail(t, err, want)
	}
	ok := ConfidenceClaim{
		ID: "a", Statement: "s", ClaimKind: ClaimKindQuestion, ConfidencePercent: 10,
		GroundTruthRef: "r", GroundTruthKind: GroundTruthHandoff, GroundTruthOutcome: GroundTruthCorrect,
	}
	if err := ValidateConfidenceClaim(ok, nil); err != nil {
		t.Fatal(err)
	}
}

func TestValidateConfidenceClaimRequiresGroundTruthForCalibrated(t *testing.T) {
	t.Parallel()
	claim := ConfidenceClaim{
		ID: "calibrated", Statement: "reviewed claim", ClaimKind: ClaimKindQuestion,
		ConfidencePercent: 90, CalibrationStatus: CalibrationCalibrated,
		SampleSize: CalibrationMinimumSample,
	}
	mustFail(t, ValidateConfidenceClaim(claim, nil), "ground_truth_ref")

	claim.GroundTruthRef = "approval_gate:1"
	claim.GroundTruthKind = GroundTruthUserRevision
	claim.GroundTruthOutcome = GroundTruthUnresolved
	mustFail(t, ValidateConfidenceClaim(claim, nil), "resolved outcome")
}

func TestValidateAssertionClaimRules(t *testing.T) {
	t.Parallel()
	base := ConfidenceClaim{ID: "a", Statement: "s", ClaimKind: ClaimKindAssertion, ConfidencePercent: 90}
	mustFail(t, ValidateConfidenceClaim(base, nil), "requires evidence")

	base.EvidenceIDs = []string{"missing"}
	base.EvidenceClasses = []string{"bogus"}
	err := ValidateConfidenceClaim(base, nil)
	mustFail(t, err, "unresolved evidence")
	mustFail(t, err, "evidence_classes[0]")

	weak := confidenceEvidence("w", "s1", EvidenceClassWeakInference)
	claim := ConfidenceClaim{ID: "a", Statement: "s", ClaimKind: ClaimKindAssertion, ConfidencePercent: 30, EvidenceIDs: []string{"w"}}
	err = ValidateConfidenceClaim(claim, []Evidence{weak})
	mustFail(t, err, "must be questions")
	mustFail(t, err, "non-supporting")

	exp := confidenceEvidence("e", "s1", EvidenceClassExplicit)
	med := ConfidenceClaim{ID: "a", Statement: "s", ClaimKind: ClaimKindAssertion, ConfidencePercent: 70, EvidenceIDs: []string{"e"}, Destructive: true}
	mustFail(t, ValidateConfidenceClaim(med, []Evidence{exp}), "cannot authorize")
	med.Destructive, med.RiskLevel = false, "high"
	mustFail(t, ValidateConfidenceClaim(med, []Evidence{exp}), "cannot authorize")

	high := ConfidenceClaim{ID: "a", Statement: "s", ClaimKind: ClaimKindAssertion, ConfidencePercent: 95, EvidenceIDs: []string{"e"}, Contradictions: []string{"c"}}
	mustFail(t, ValidateConfidenceClaim(high, []Evidence{exp}), "contradictions")
	high.Contradictions = nil
	if err := ValidateConfidenceClaim(high, []Evidence{exp}); err != nil {
		t.Fatal(err)
	}
	c1 := confidenceEvidence("c1", "same", EvidenceClassCorroboratedInference)
	c2 := confidenceEvidence("c2", "same", EvidenceClassCorroboratedInference)
	high.EvidenceIDs = []string{"c1", "c2"}
	mustFail(t, ValidateConfidenceClaim(high, []Evidence{c1, c2}), "two independent")
	c2.SourceRef = "other"
	if err := ValidateConfidenceClaim(high, []Evidence{c1, c2}); err != nil {
		t.Fatal(err)
	}
}

func validSummary() ConfidenceSummary {
	ev := confidenceEvidence("e", "s", EvidenceClassExplicit)
	return ConfidenceSummary{
		PolicyVersion: ConfidencePolicyVersion, CalibrationStatus: CalibrationNoSample,
		Evidence: []Evidence{ev},
		Claims: []ConfidenceClaim{{
			ID: "c1", Statement: "s", Agent: "ranger", CorrelationKey: "k1", ClaimKind: ClaimKindAssertion,
			ConfidencePercent: 90, EvidenceIDs: []string{"e"},
		}},
		OpenQuestions: []ConfidenceClaim{{
			ID: "q1", Statement: "?", Agent: "ranger", CorrelationKey: "k2", ClaimKind: ClaimKindQuestion, ConfidencePercent: 20,
		}},
	}
}

func TestValidateConfidenceSummary(t *testing.T) {
	t.Parallel()
	if err := ValidateConfidenceSummary(validSummary()); err != nil {
		t.Fatal(err)
	}
	s := validSummary()
	s.PolicyVersion, s.SampleSize = "v0", -1
	err := ValidateConfidenceSummary(s)
	mustFail(t, err, "policy_version")
	mustFail(t, err, "sample_size cannot be negative")

	s = ConfidenceSummary{PolicyVersion: ConfidencePolicyVersion, CalibrationStatus: CalibrationNoSample}
	mustFail(t, ValidateConfidenceSummary(s), "requires missing_record")

	s.MissingRecord = true
	if err := ValidateConfidenceSummary(s); err != nil {
		t.Fatal(err)
	}
	s = validSummary()
	s.MissingRecord, s.SampleSize, s.CalibrationStatus = true, 3, CalibrationObserved
	err = ValidateConfidenceSummary(s)
	mustFail(t, err, "cannot carry")
	mustFail(t, err, "requires sample_size 0")

	s = validSummary()
	s.Evidence = append(s.Evidence, s.Evidence[0], Evidence{})
	mustFail(t, ValidateConfidenceSummary(s), "duplicate evidence id")

	s = validSummary()
	s.Claims[0].Agent, s.Claims[0].CorrelationKey = "", ""
	s.OpenQuestions[0].ClaimKind = ClaimKindAssertion
	s.Claims = append(s.Claims, s.Claims[0])
	err = ValidateConfidenceSummary(s)
	mustFail(t, err, "requires agent")
	mustFail(t, err, "requires correlation_key")
	mustFail(t, err, "must have claim_kind question")
	mustFail(t, err, "duplicate claim id")
}

func TestCompareConfidenceSummaries(t *testing.T) {
	t.Parallel()
	src := validSummary()
	if err := CompareConfidenceSummaries(src, validSummary()); err != nil {
		t.Fatal(err)
	}
	bad := validSummary()
	bad.PolicyVersion = "x"
	mustFail(t, CompareConfidenceSummaries(bad, validSummary()), "source confidence summary")
	mustFail(t, CompareConfidenceSummaries(validSummary(), bad), "destination confidence summary")

	missing := ConfidenceSummary{PolicyVersion: ConfidencePolicyVersion, CalibrationStatus: CalibrationNoSample, MissingRecord: true}
	if err := CompareConfidenceSummaries(missing, missing); err != nil {
		t.Fatal(err)
	}

	dst := validSummary()
	dst.OpenQuestions = nil
	mustFail(t, CompareConfidenceSummaries(src, dst), "was not preserved")

	dst = validSummary()
	dst.Claims[0].ClaimKind = ClaimKindQuestion
	dst.Claims[0].ConfidencePercent = 50
	dst.Claims[0].EvidenceIDs = nil
	dst.Claims[0].EvidenceClasses = nil
	err := CompareConfidenceSummaries(src, dst)
	mustFail(t, err, "changed claim_kind")
	mustFail(t, err, "changed confidence")
	mustFail(t, err, "changed evidence_ids")

	src2, dst2 := validSummary(), validSummary()
	src2.Claims[0].EvidenceClasses = []string{EvidenceClassExplicit}
	dst2.Claims[0].EvidenceClasses = []string{EvidenceClassCorroboratedInference}
	dst2.Claims[0].GroundTruthRef, dst2.Claims[0].GroundTruthKind = "r", GroundTruthHandoff
	err = CompareConfidenceSummaries(src2, dst2)
	mustFail(t, err, "changed evidence_classes")
	mustFail(t, err, "changed ground truth")

	dst2 = validSummary()
	dst2.Claims[0].Statement = "rewritten"
	err = CompareConfidenceSummaries(src2, dst2)
	mustFail(t, err, "changed statement")

	dst2 = validSummary()
	dst2.Claims[0].CalibrationStatus = CalibrationObserved
	dst2.Claims[0].SampleSize = 1
	err = CompareConfidenceSummaries(src2, dst2)
	mustFail(t, err, "changed calibration")
	if !sameStrings([]string{"a"}, []string{"a"}) || sameStrings([]string{"a"}, []string{"b"}) {
		t.Fatal("sameStrings")
	}
}

func TestValidateDecisionClaimBranches(t *testing.T) {
	t.Parallel()
	pct := 90
	d := Decision{ID: "d", Statement: "s", Status: DecisionStatusOpen, Confidence: ConfidenceHigh, ClaimKind: "weird"}
	err := ValidateDecision(d)
	mustFail(t, err, "requires confidence_percent")
	mustFail(t, err, "claim_kind")
	d.ClaimKind, d.ConfidencePercent = ClaimKindAssertion, &pct
	d.Confidence = ConfidenceLow
	mustFail(t, ValidateDecision(d), "conflicts")

	d = Decision{ID: "d", Statement: "s", Status: DecisionStatusOpen, Confidence: ConfidenceHigh, ConfidencePercent: &pct}
	mustFail(t, ValidateDecisionWithEvidence(d, nil), "")
	d.Confidence = ""
	if err := ValidateDecisionWithEvidence(Decision{ID: "d", Statement: "s", Status: DecisionStatusOpen, Confidence: ConfidenceLow}, nil); err != nil {
		t.Fatal(err)
	}
	d = Decision{ID: "d", Statement: "s", Status: DecisionStatusOpen, Confidence: ConfidenceHigh, ClaimKind: ClaimKindAssertion, ConfidencePercent: &pct, EvidenceIDs: []string{"e"}}
	if err := ValidateDecisionWithEvidence(d, []Evidence{confidenceEvidence("e", "s", EvidenceClassExplicit)}); err != nil {
		t.Fatal(err)
	}
	mustFail(t, ValidateDecisionWithEvidence(Decision{}, nil), "decision_invalid")
}

func TestFindCatalogRankedStampAndPhaseError(t *testing.T) {
	t.Parallel()
	if _, _, err := FindCatalogRankedStamp([]byte(":\n\t- x"), "p"); err == nil {
		t.Fatal("expected parse error")
	}
	if _, found, err := FindCatalogRankedStamp([]byte("providers: []\n"), "p"); err != nil || found {
		t.Fatalf("found=%v err=%v", found, err)
	}
	if s, found, err := FindCatalogRankedStamp([]byte("providers:\n  - id: p\n    ranked: true\n"), "p"); err != nil || !found || s.ID != "p" {
		t.Fatalf("stamp=%+v found=%v err=%v", s, found, err)
	}
	msg := ErrOutOfOrderPhaseSubmit{Current: PhaseBootstrap, Event: PhaseEvent("x")}.Error()
	if !strings.Contains(msg, "not valid from phase") {
		t.Fatal(msg)
	}
}

package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func confidenceEvidence(id, source, class string) Evidence {
	return Evidence{ID: id, SourceRef: source, Class: class, Confidence: ConfidenceHigh}
}

func TestConfidenceLevelForPercentBoundaries(t *testing.T) {
	t.Parallel()
	cases := []struct {
		percent int
		level   string
	}{
		{0, ConfidenceLow}, {59, ConfidenceLow}, {60, ConfidenceMedium},
		{84, ConfidenceMedium}, {85, ConfidenceHigh}, {100, ConfidenceHigh},
	}
	for _, tc := range cases {
		level, err := ConfidenceLevelForPercent(tc.percent)
		if err != nil || level != tc.level {
			t.Fatalf("percent %d: level=%q err=%v, want %q", tc.percent, level, err, tc.level)
		}
	}
}

func TestConfidenceLevelForPercentRejectsOutOfRange(t *testing.T) {
	t.Parallel()
	for _, percent := range []int{-1, 101} {
		if _, err := ConfidenceLevelForPercent(percent); err == nil {
			t.Fatalf("expected percent %d to be rejected", percent)
		}
	}
}

func TestValidateConfidenceClaimQuestionPreservesUncertainty(t *testing.T) {
	t.Parallel()
	claim := ConfidenceClaim{ID: "Q-001", Statement: "Could this be stale?", ClaimKind: ClaimKindQuestion, ConfidencePercent: 40}
	if err := ValidateConfidenceClaim(claim, nil); err != nil {
		t.Fatalf("question should be valid without evidence: %v", err)
	}
}

func TestValidateConfidenceClaimRejectsUnsupportedAssertion(t *testing.T) {
	t.Parallel()
	claim := ConfidenceClaim{ID: "A-001", Statement: "It is fixed.", ClaimKind: ClaimKindAssertion, ConfidencePercent: 75}
	if err := ValidateConfidenceClaim(claim, nil); err == nil || !strings.Contains(err.Error(), "requires evidence") {
		t.Fatalf("expected unsupported assertion error, got %v", err)
	}
}

func TestValidateConfidenceClaimRejectsLowAssertion(t *testing.T) {
	t.Parallel()
	claim := ConfidenceClaim{ID: "A-002", Statement: "It is fixed.", ClaimKind: ClaimKindAssertion, ConfidencePercent: 20, EvidenceIDs: []string{"E-001"}}
	evidence := []Evidence{confidenceEvidence("E-001", "test.go", EvidenceClassExplicit)}
	if err := ValidateConfidenceClaim(claim, evidence); err == nil || !strings.Contains(err.Error(), "low-confidence") {
		t.Fatalf("expected low assertion error, got %v", err)
	}
}

func TestValidateConfidenceClaimRejectsMediumHighRisk(t *testing.T) {
	t.Parallel()
	claim := ConfidenceClaim{ID: "A-003", Statement: "Deploy it.", ClaimKind: ClaimKindAssertion, ConfidencePercent: 70, RiskLevel: "high", EvidenceIDs: []string{"E-001"}}
	evidence := []Evidence{confidenceEvidence("E-001", "test.go", EvidenceClassExplicit)}
	if err := ValidateConfidenceClaim(claim, evidence); err == nil || !strings.Contains(err.Error(), "high-risk") {
		t.Fatalf("expected risk error, got %v", err)
	}
}

func TestValidateConfidenceClaimAcceptsMediumProvisionalAssertion(t *testing.T) {
	t.Parallel()
	claim := ConfidenceClaim{ID: "A-003b", Statement: "The patch is ready for review.", ClaimKind: ClaimKindAssertion, ConfidencePercent: 70, EvidenceIDs: []string{"E-001"}}
	evidence := []Evidence{confidenceEvidence("E-001", "test.go", EvidenceClassExplicit)}
	if err := ValidateConfidenceClaim(claim, evidence); err != nil {
		t.Fatalf("medium provisional assertion should pass with explicit evidence: %v", err)
	}
}

func TestValidateConfidenceClaimRejectsWeakInferenceAssertion(t *testing.T) {
	t.Parallel()
	claim := ConfidenceClaim{ID: "A-003c", Statement: "It is probably fixed.", ClaimKind: ClaimKindAssertion, ConfidencePercent: 70, EvidenceIDs: []string{"E-001"}}
	evidence := []Evidence{confidenceEvidence("E-001", "test.go", EvidenceClassWeakInference)}
	if err := ValidateConfidenceClaim(claim, evidence); err == nil || !strings.Contains(err.Error(), "non-supporting class") {
		t.Fatalf("expected weak inference to be rejected, got %v", err)
	}
}

func TestValidateConfidenceClaimHighNeedsExplicitOrIndependentCorroboration(t *testing.T) {
	t.Parallel()
	claim := ConfidenceClaim{ID: "A-004", Statement: "It is fixed.", ClaimKind: ClaimKindAssertion, ConfidencePercent: 90, EvidenceIDs: []string{"E-001"}}
	oneSource := []Evidence{confidenceEvidence("E-001", "same.go", EvidenceClassCorroboratedInference)}
	if err := ValidateConfidenceClaim(claim, oneSource); err == nil || !strings.Contains(err.Error(), "two independent") {
		t.Fatalf("expected corroboration error, got %v", err)
	}
	claim.EvidenceIDs = []string{"E-001", "E-002"}
	twoSources := make([]Evidence, len(oneSource), len(oneSource)+1)
	copy(twoSources, oneSource)
	twoSources = append(twoSources, confidenceEvidence("E-002", "other.go", EvidenceClassCorroboratedInference))
	if err := ValidateConfidenceClaim(claim, twoSources); err != nil {
		t.Fatalf("independent corroboration should pass: %v", err)
	}
}

func TestValidateConfidenceClaimRejectsContradictoryHighAssertion(t *testing.T) {
	t.Parallel()
	claim := ConfidenceClaim{ID: "A-005", Statement: "It is fixed.", ClaimKind: ClaimKindAssertion, ConfidencePercent: 90, Contradictions: []string{"E-999"}, EvidenceIDs: []string{"E-001"}}
	evidence := []Evidence{confidenceEvidence("E-001", "test.go", EvidenceClassExplicit)}
	if err := ValidateConfidenceClaim(claim, evidence); err == nil || !strings.Contains(err.Error(), "contradictions") {
		t.Fatalf("expected contradictory high assertion to be rejected, got %v", err)
	}
}

func TestValidateCalibrationStatusEmptySampleIsNoSample(t *testing.T) {
	t.Parallel()
	if err := ValidateCalibrationStatus(CalibrationUncalibrated, 0); err == nil {
		t.Fatal("expected non-no_sample status to be rejected for empty sample")
	}
	if err := ValidateCalibrationStatus(CalibrationNoSample, 0); err != nil {
		t.Fatalf("no_sample should be valid: %v", err)
	}
}

func TestDecisionAndEvidenceRejectConflictingPercentage(t *testing.T) {
	t.Parallel()
	percent := 70
	d := validDecision()
	d.ConfidencePercent = &percent
	if err := ValidateDecision(d); err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("expected decision conflict, got %v", err)
	}
	e := validEvidence()
	e.ConfidencePercent = &percent
	if err := ValidateEvidence(e); err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("expected evidence conflict, got %v", err)
	}
}

func TestConfidenceClaimUsesCanonicalEvidenceIDsAndReadsLegacyAlias(t *testing.T) {
	t.Parallel()
	claim := ConfidenceClaim{ID: "A-1", Statement: "supported", ClaimKind: ClaimKindAssertion, ConfidencePercent: 70, EvidenceIDs: []string{"E-1"}}
	encoded, err := json.Marshal(claim)
	if err != nil {
		t.Fatalf("marshal claim: %v", err)
	}
	if strings.Contains(string(encoded), `"evidence":`) || !strings.Contains(string(encoded), `"evidence_ids"`) {
		t.Fatalf("claim was not emitted canonically: %s", encoded)
	}
	var legacy ConfidenceClaim
	if err := json.Unmarshal([]byte(`{"id":"A-1","statement":"supported","claim_kind":"assertion","confidence_percent":70,"evidence":["E-1"]}`), &legacy); err != nil {
		t.Fatalf("legacy alias should be readable: %v", err)
	}
	if len(legacy.EvidenceIDs) != 1 || legacy.EvidenceIDs[0] != "E-1" {
		t.Fatalf("legacy evidence was not normalized: %+v", legacy)
	}
	var conflicting ConfidenceClaim
	if err := json.Unmarshal([]byte(`{"id":"A-1","statement":"supported","claim_kind":"assertion","confidence_percent":70,"evidence":["E-1"],"evidence_ids":["E-2"]}`), &conflicting); err == nil {
		t.Fatal("conflicting evidence aliases should fail")
	}
}

func TestValidateCalibrationStatusRequiresReviewedMinimumForCalibrated(t *testing.T) {
	t.Parallel()
	if err := ValidateCalibrationStatus(CalibrationCalibrated, CalibrationMinimumSample-1); err == nil {
		t.Fatal("expected calibrated status below minimum sample to fail")
	}
}

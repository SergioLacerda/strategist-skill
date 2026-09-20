package domain

import (
	"strings"
	"testing"
)

func validConfidenceSummary() ConfidenceSummary {
	return ConfidenceSummary{
		PolicyVersion: ConfidencePolicyVersion,
		Claims: []ConfidenceClaim{{
			ID: "A-1", Statement: "The evidence is explicit.", Agent: "ranger",
			CorrelationKey: "q-1", ClaimKind: ClaimKindAssertion, ConfidencePercent: 90,
			EvidenceIDs: []string{"E-1"}, EvidenceClasses: []string{EvidenceClassExplicit},
			SampleSize: 0, CalibrationStatus: CalibrationNoSample,
		}},
		Evidence:   []Evidence{{ID: "E-1", SourceRef: "docs/finding.md", Class: EvidenceClassExplicit, Confidence: ConfidenceHigh}},
		SampleSize: 0, CalibrationStatus: CalibrationNoSample,
	}
}

func TestValidateConfidenceSummaryRequiresCorrelationAndAgent(t *testing.T) {
	t.Parallel()
	summary := validConfidenceSummary()
	summary.Claims[0].Agent = ""
	summary.Claims[0].CorrelationKey = ""
	err := ValidateConfidenceSummary(summary)
	if err == nil || !strings.Contains(err.Error(), "requires agent") || !strings.Contains(err.Error(), "requires correlation_key") {
		t.Fatalf("expected agent and correlation errors, got %v", err)
	}
}

func TestCompareConfidenceSummariesRejectsDroppedQuestion(t *testing.T) {
	t.Parallel()
	source := validConfidenceSummary()
	destination := source
	destination.Claims = nil
	destination.Evidence = nil
	destination.MissingRecord = true
	if err := CompareConfidenceSummaries(source, destination); err == nil || !strings.Contains(err.Error(), "was not preserved") {
		t.Fatalf("expected dropped claim error, got %v", err)
	}
}

func TestValidateConfidenceSummaryMissingRecordIsExplicit(t *testing.T) {
	t.Parallel()
	summary := ConfidenceSummary{PolicyVersion: ConfidencePolicyVersion, MissingRecord: true, CalibrationStatus: CalibrationNoSample}
	if err := ValidateConfidenceSummary(summary); err != nil {
		t.Fatalf("valid missing summary: %v", err)
	}
	summary.Claims = []ConfidenceClaim{{ID: "unexpected"}}
	if err := ValidateConfidenceSummary(summary); err == nil {
		t.Fatal("expected missing summary with claims to fail")
	}
}

package domain

import (
	"strings"
	"testing"
)

func validDecision() Decision {
	return Decision{
		ID:          "DEC-001",
		Statement:   "Use write_analysis for Ranger.",
		Status:      DecisionStatusApproved,
		EvidenceIDs: []string{"EVD-001"},
		Confidence:  ConfidenceHigh,
	}
}

func TestValidateDecision_Valid(t *testing.T) {
	t.Parallel()
	if err := ValidateDecision(validDecision()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateDecision_MissingFields(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		mutate  func(d Decision) Decision
		missing string
	}{
		{"missing id", func(d Decision) Decision { d.ID = ""; return d }, "id is required"},
		{"missing statement", func(d Decision) Decision { d.Statement = ""; return d }, "statement is required"},
		{"missing status", func(d Decision) Decision { d.Status = ""; return d }, "status is required"},
		{"missing confidence", func(d Decision) Decision { d.Confidence = ""; return d }, "confidence is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateDecision(tc.mutate(validDecision()))
			if err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.missing) {
				t.Fatalf("error should mention %q: %v", tc.missing, err)
			}
		})
	}
}

func TestValidateDecision_InvalidValues(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		mutate func(d Decision) Decision
	}{
		{"bad status", func(d Decision) Decision { d.Status = "maybe"; return d }},
		{"bad confidence", func(d Decision) Decision { d.Confidence = "extreme"; return d }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := ValidateDecision(tc.mutate(validDecision())); err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
		})
	}
}

func TestValidateDecision_EmptyEvidenceListIsStructurallyValid(t *testing.T) {
	t.Parallel()
	// An empty evidence list is a mission_quality concern
	// (checkUnsupportedClaims), not a structural validation error here —
	// ValidateDecision only checks required scalar fields and allowed
	// enum values.
	d := validDecision()
	d.EvidenceIDs = nil
	if err := ValidateDecision(d); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAllDecisionStatuses_AreAccepted(t *testing.T) {
	t.Parallel()
	for _, status := range []string{DecisionStatusOpen, DecisionStatusApproved, DecisionStatusRejected, DecisionStatusSuperseded} {
		d := validDecision()
		d.Status = status
		if err := ValidateDecision(d); err != nil {
			t.Fatalf("status %q should be valid: %v", status, err)
		}
	}
}

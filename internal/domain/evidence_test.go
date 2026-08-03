package domain

import (
	"strings"
	"testing"
)

func validEvidence() Evidence {
	return Evidence{
		ID:         "EVD-001",
		SourceRef:  "docs/adr/0005-slot-write-contracts.md",
		Class:      EvidenceClassExplicit,
		Confidence: ConfidenceHigh,
	}
}

func TestValidateEvidence_Valid(t *testing.T) {
	t.Parallel()
	if err := ValidateEvidence(validEvidence()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateEvidence_MissingFields(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		mutate  func(e Evidence) Evidence
		missing string
	}{
		{"missing id", func(e Evidence) Evidence { e.ID = ""; return e }, "id is required"},
		{"missing source_ref", func(e Evidence) Evidence { e.SourceRef = ""; return e }, "source_ref is required"},
		{"missing class", func(e Evidence) Evidence { e.Class = ""; return e }, "class is required"},
		{"missing confidence", func(e Evidence) Evidence { e.Confidence = ""; return e }, "confidence is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateEvidence(tc.mutate(validEvidence()))
			if err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.missing) {
				t.Fatalf("error should mention %q: %v", tc.missing, err)
			}
		})
	}
}

func TestValidateEvidence_InvalidValues(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		mutate func(e Evidence) Evidence
	}{
		{"bad class", func(e Evidence) Evidence { e.Class = "guess"; return e }},
		{"bad confidence", func(e Evidence) Evidence { e.Confidence = "extreme"; return e }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := ValidateEvidence(tc.mutate(validEvidence())); err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
		})
	}
}

func TestAllEvidenceClasses_AreAccepted(t *testing.T) {
	t.Parallel()
	classes := []string{
		EvidenceClassExplicit,
		EvidenceClassCorroboratedInference,
		EvidenceClassWeakInference,
		EvidenceClassUnknown,
	}
	for _, class := range classes {
		e := validEvidence()
		e.Class = class
		if err := ValidateEvidence(e); err != nil {
			t.Fatalf("class %q should be valid: %v", class, err)
		}
	}
}

func TestValidateEvidence_ValidUntilIsOptional(t *testing.T) {
	t.Parallel()
	e := validEvidence()
	e.ValidUntil = ""
	if err := ValidateEvidence(e); err != nil {
		t.Fatalf("unexpected error with empty ValidUntil: %v", err)
	}
}

package runbook

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

func TestValidateEvidenceRef_Valid(t *testing.T) {
	t.Parallel()
	err := ValidateEvidenceRef(EvidenceRef{ID: "EVD-001", Class: domain.EvidenceClassExplicit, Confidence: domain.ConfidenceHigh})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateEvidenceRef_InvalidClassOrConfidence(t *testing.T) {
	t.Parallel()
	cases := []EvidenceRef{
		{ID: "EVD-001", Class: "bogus", Confidence: domain.ConfidenceHigh},
		{ID: "EVD-001", Class: domain.EvidenceClassExplicit, Confidence: "bogus"},
		{Class: domain.EvidenceClassExplicit, Confidence: domain.ConfidenceHigh},
	}
	for _, e := range cases {
		if err := ValidateEvidenceRef(e); err == nil {
			t.Errorf("expected error for %+v", e)
		}
	}
}

func TestEvidenceRef_Qualifies(t *testing.T) {
	t.Parallel()
	cases := []struct {
		class     string
		qualifies bool
	}{
		{domain.EvidenceClassExplicit, true},
		{domain.EvidenceClassCorroboratedInference, true},
		{domain.EvidenceClassWeakInference, false},
		{domain.EvidenceClassUnknown, false},
	}
	for _, c := range cases {
		got := EvidenceRef{Class: c.class}.qualifies()
		if got != c.qualifies {
			t.Errorf("class %q: qualifies() = %v, want %v", c.class, got, c.qualifies)
		}
	}
}

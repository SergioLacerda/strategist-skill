package runbook

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

func TestEvaluateStep_SatisfiedByQualifyingEvidence(t *testing.T) {
	t.Parallel()
	check := Check{ID: "reproduced-locally", Level: LevelMandatory}
	evidence := []EvidenceRef{{ID: "EVD-001", Class: domain.EvidenceClassExplicit, Confidence: domain.ConfidenceHigh}}
	result := EvaluateStep(check, evidence, "")
	if !result.Satisfied || result.Excepted {
		t.Fatalf("expected satisfied without exception, got %+v", result)
	}
}

func TestEvaluateStep_WeakEvidenceAloneDoesNotSatisfy(t *testing.T) {
	t.Parallel()
	check := Check{ID: "reproduced-locally", Level: LevelMandatory}
	evidence := []EvidenceRef{{ID: "EVD-001", Class: domain.EvidenceClassWeakInference, Confidence: domain.ConfidenceLow}}
	result := EvaluateStep(check, evidence, "")
	if result.Satisfied {
		t.Fatalf("expected unsatisfied without a justification, got %+v", result)
	}
}

func TestEvaluateStep_JustificationSatisfiesAsException(t *testing.T) {
	t.Parallel()
	check := Check{ID: "reproduced-locally", Level: LevelMandatory}
	result := EvaluateStep(check, nil, "cannot reproduce outside prod, accepted risk per on-call lead")
	if !result.Satisfied || !result.Excepted {
		t.Fatalf("expected satisfied via exception, got %+v", result)
	}
}

func TestEvaluateStep_NoEvidenceNoJustificationIsUnsatisfied(t *testing.T) {
	t.Parallel()
	check := Check{ID: "reproduced-locally", Level: LevelMandatory}
	result := EvaluateStep(check, nil, "")
	if result.Satisfied {
		t.Fatalf("expected unsatisfied, got %+v", result)
	}
}

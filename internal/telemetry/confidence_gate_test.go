package telemetry

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

func TestBuildConfidenceGateReviewFailsClosedOnUnavailableEvidence(t *testing.T) {
	t.Parallel()
	review := BuildConfidenceGateReview(nil, ConfidenceReadDiagnostics{})
	if !review.ReviewRequired || !review.Unavailable {
		t.Fatalf("empty confidence evidence must require review: %+v", review)
	}
	if len(review.Reasons) == 0 || review.Reasons[0] != "confidence_evidence_unavailable" {
		t.Fatalf("unexpected reasons: %v", review.Reasons)
	}
}

func TestBuildConfidenceGateReviewMarksLowAndUncalibratedMaterial(t *testing.T) {
	t.Parallel()
	record := confidenceRecordFixture("q-1", "ranger", domain.ClaimKindQuestion, domain.ConfidenceLow, 20)
	record.CalibrationStatus = domain.CalibrationUncalibrated
	review := BuildConfidenceGateReview([]ConfidenceRecord{record}, ConfidenceReadDiagnostics{})
	if !review.ReviewRequired {
		t.Fatal("low confidence must remain a review signal")
	}
}

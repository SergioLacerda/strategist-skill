package telemetry

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// ConfidenceGateReview is an advisory projection consumed by the human gate.
// It intentionally has no approval or execution field: confidence can request
// review, but cannot authorize the next phase.
type ConfidenceGateReview struct {
	Metrics        ConfidenceMetrics         `json:"metrics"`
	Diagnostics    ConfidenceReadDiagnostics `json:"diagnostics"`
	ReviewRequired bool                      `json:"review_required"`
	Unavailable    bool                      `json:"unavailable"`
	Reasons        []string                  `json:"reasons,omitempty"`
}

// BuildConfidenceGateReview projects confidence history into the advisory
// information shown alongside the human approval gate.
func BuildConfidenceGateReview(records []ConfidenceRecord, diagnostics ConfidenceReadDiagnostics) ConfidenceGateReview {
	metrics := ComputeConfidenceMetrics(records)
	review := ConfidenceGateReview{Metrics: metrics, Diagnostics: diagnostics}
	review.Metrics.RejectedRecords += diagnostics.MalformedLines + diagnostics.InvalidRecords
	review.Metrics.DuplicateRecords += diagnostics.DuplicateEvents
	applyReviewDiagnostics(&review, records, metrics, diagnostics)
	applyReviewRecordFlags(&review, records)
	applyReviewMetricFlags(&review, metrics)
	return review
}

func applyReviewDiagnostics(review *ConfidenceGateReview, records []ConfidenceRecord, metrics ConfidenceMetrics, diagnostics ConfidenceReadDiagnostics) {
	if len(records) == 0 || metrics.SampleSize == 0 {
		review.ReviewRequired = true
		review.Unavailable = true
		review.Reasons = append(review.Reasons, "confidence_evidence_unavailable")
	}
	if metrics.MissingRecords > 0 {
		review.ReviewRequired = true
		review.Reasons = append(review.Reasons, fmt.Sprintf("missing_confidence_records:%d", metrics.MissingRecords))
	}
	if metrics.RejectedRecords > 0 || diagnostics.MalformedLines > 0 || diagnostics.InvalidRecords > 0 {
		review.ReviewRequired = true
		review.Reasons = append(review.Reasons, "invalid_confidence_records")
	}
	if metrics.DuplicateRecords > 0 || diagnostics.DuplicateEvents > 0 {
		review.ReviewRequired = true
		review.Reasons = append(review.Reasons, "duplicate_confidence_records")
	}
}

func applyReviewRecordFlags(review *ConfidenceGateReview, records []ConfidenceRecord) {
	for _, record := range records {
		if record.ConfidenceLevel == domain.ConfidenceLow || record.CoverageStatus == ConfidenceCoverageRejected || record.Violation != "" {
			review.ReviewRequired = true
		}
	}
}

func applyReviewMetricFlags(review *ConfidenceGateReview, metrics ConfidenceMetrics) {
	if metrics.UnsupportedAssertionRate > 0 {
		review.ReviewRequired = true
		review.Reasons = append(review.Reasons, "unsupported_assertions")
	}
	if metrics.CalibrationStatus != domain.CalibrationCalibrated {
		review.RequiredCalibrationReason()
	}
}

// RequiredCalibrationReason keeps calibration advisory and visible without
// turning an uncalibrated sample into an automatic rejection.
func (r *ConfidenceGateReview) RequiredCalibrationReason() {
	if r == nil {
		return
	}
	r.ReviewRequired = true
	r.Reasons = append(r.Reasons, "calibration_not_established")
}

package missionview

import "github.com/SergioLacerda/strategist-skill/internal/telemetry"

func selection(run string) string {
	if run != "" {
		return "selected_run"
	}
	return "latest_record"
}

func confidenceAvailability(review telemetry.ConfidenceGateReview) string {
	if review.Metrics.SampleSize == 0 {
		return NoSample
	}
	return Available
}

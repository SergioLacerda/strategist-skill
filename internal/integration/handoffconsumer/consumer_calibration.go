package handoffconsumer

import "github.com/SergioLacerda/strategist-skill/internal/integration"

// appliedID is the calibration id only when that calibration decided the threshold.
func appliedID(use CalibrationUse) string {
	if use.Applied {
		return use.ID
	}
	return ""
}

// withProbability drops a calibration when the answer has no probability of accept: a
// calibrated threshold is a threshold on that quantity, and the documented confidence
// is a different one, so the configured threshold applies and the reason is recorded.
func withProbability(use CalibrationUse, result integration.Result) CalibrationUse {
	if !use.Applied {
		return use
	}
	if _, reported := result.Answers[AggregateKey].Probabilities[OptionAccept]; reported {
		return use
	}
	return CalibrationUse{ID: use.ID, Reason: ReasonProbabilityUnavailable}
}

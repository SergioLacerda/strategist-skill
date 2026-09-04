package telemetry

import "github.com/SergioLacerda/strategist-skill/internal/domain"

// FallbackMetrics reports how often each provider_resolution_policy outcome
// (ADR-0028) accounted for a recorded provider-fallback degradation event.
// Every FallbackDecision in the sample already represents an applied
// fallback (see FallbackDecision's own doc comment — blocked/unavailable
// outcomes are never recorded), so AutoNativeRate + AskConfirmedRate always
// sum to 1 for a non-empty sample.
type FallbackMetrics struct {
	AutoNativeRate   float64
	AskConfirmedRate float64
	SampleSize       int
}

// ComputeFallbackMetrics aggregates decisions into FallbackMetrics. Both
// rates are 0 when the sample is empty, rather than NaN, so callers can
// render "0.00" instead of special-casing an empty history.
func ComputeFallbackMetrics(decisions []FallbackDecision) FallbackMetrics {
	var autoNative, askConfirmed int
	for _, d := range decisions {
		switch d.Outcome {
		case string(domain.FallbackOutcomeAutoNative):
			autoNative++
		case string(domain.FallbackOutcomeAskRequired):
			askConfirmed++
		}
	}
	total := len(decisions)
	return FallbackMetrics{
		AutoNativeRate:   safeRate(autoNative, total),
		AskConfirmedRate: safeRate(askConfirmed, total),
		SampleSize:       total,
	}
}

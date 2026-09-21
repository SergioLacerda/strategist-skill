package telemetry

import "github.com/SergioLacerda/strategist-skill/internal/domain"

// RouteGroundTruthMetrics are the reversal-dependent Scout metrics. Every
// rate is 0 with SampleSize 0 (reported as no_sample) when no decision has a
// reviewed label; rates are never inferred from unlabeled decisions.
type RouteGroundTruthMetrics struct {
	RouteAccuracy               float64
	DirectRouteReversalRate     float64
	RiskUnderclassificationRate float64
	UserOverrideRate            float64
	SampleSize                  int
	DirectRouteSampleSize       int
	CalibrationStatus           string
}

// routeTally accumulates labeled-decision counts for one metrics pass.
type routeTally struct {
	sample, directSample, confirmed, directReversed, underclassified, overridden int
}

func (t *routeTally) add(direct bool, label string) {
	t.sample++
	if direct {
		t.directSample++
	}
	switch label {
	case RouteLabelConfirmed:
		t.confirmed++
	case RouteLabelReversed:
		t.directReversed += boolToInt(direct)
	case RouteLabelRiskUnderclassified:
		t.underclassified++
	case RouteLabelUserOverride:
		t.overridden++
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ComputeRouteGroundTruthMetrics joins decisions with route labels by
// mission_id. Direct routes are every route other than full_pipeline.
func ComputeRouteGroundTruthMetrics(decisions []RouteDecision, labels []GroundTruthLabel) RouteGroundTruthMetrics {
	labelByMission := map[string]string{}
	for _, l := range labels {
		if l.Subject == GroundTruthSubjectRoute {
			labelByMission[l.MissionID] = l.Label
		}
	}
	var t routeTally
	for _, d := range decisions {
		if label, ok := labelByMission[d.MissionID]; ok {
			t.add(d.SelectedRoute != "full_pipeline", label)
		}
	}
	return RouteGroundTruthMetrics{
		RouteAccuracy:               safeRate(t.confirmed, t.sample),
		DirectRouteReversalRate:     safeRate(t.directReversed, t.directSample),
		RiskUnderclassificationRate: safeRate(t.underclassified, t.sample),
		UserOverrideRate:            safeRate(t.overridden, t.sample),
		SampleSize:                  t.sample,
		DirectRouteSampleSize:       t.directSample,
		CalibrationStatus:           calibrationStatusForSample(t.sample),
	}
}

// calibrationStatusForSample keeps an empty label history distinct from a
// measured 0%, and small samples distinct from a calibrated one.
func calibrationStatusForSample(sample int) string {
	switch {
	case sample == 0:
		return domain.CalibrationNoSample
	case sample < domain.CalibrationMinimumSample:
		return domain.CalibrationUncalibrated
	default:
		return domain.CalibrationObserved
	}
}

// ApplyHandoffApplicationGroundTruth fills SemanticLoss.Application from
// downstream application labels: 1 - applied/labeled. With no labels it stays
// 0 and ApplicationSampleSize stays 0 (no_sample), never a claim of success.
func ApplyHandoffApplicationGroundTruth(m HandoffMetrics, labels []GroundTruthLabel) HandoffMetrics {
	var sample, applied int
	for _, l := range labels {
		if l.Subject != GroundTruthSubjectHandoffApplication {
			continue
		}
		sample++
		if l.Label == HandoffApplicationLabelApplied {
			applied++
		}
	}
	m.ApplicationSampleSize = sample
	if sample > 0 {
		m.SemanticLoss.Application = 1 - safeRate(applied, sample)
	}
	return m
}

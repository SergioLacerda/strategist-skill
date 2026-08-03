package telemetry

// RouteMetrics reports the Phase-1 Scout routing metrics computable without a
// reversal-labeling mechanism (see
// .analysis/refined/20260803-scout-routing-metrics/design.md § D2/D3):
// FallbackRate and UnnecessaryPipelineRate. The four reversal-dependent
// metrics (route_accuracy, direct_route_reversal_rate,
// risk_underclassification_rate, user_override_rate) are out of scope for
// this type — they need a ground-truth labeling source that does not exist
// yet.
type RouteMetrics struct {
	FallbackRate            float64
	UnnecessaryPipelineRate float64
	SampleSize              int
	FullPipelineSampleSize  int
}

// analysisDeliveredStatus is the mission_status vocabulary value a mission
// resolves to when Archivist produced no documentation_targets — see
// contracts/machine/mission-status.yaml's archivist_done note and
// contracts/narrative/04-refinement.md's Gate Condition. It is used here as
// the proxy signal for "this full_pipeline mission turned out not to need
// the pipeline" (unnecessary_pipeline_rate), avoiding a second filesystem
// read of the mission's tasks.md.
const analysisDeliveredStatus = "analysis_delivered"

// ComputeRouteMetrics aggregates decisions (and, for
// UnnecessaryPipelineRate, the matching outcomes by mission_id) into
// RouteMetrics. Both rates are 0 when their respective sample is empty,
// rather than NaN, so callers can render "0.00" instead of special-casing an
// empty history.
func ComputeRouteMetrics(decisions []RouteDecision, outcomes []OutcomeEntry) RouteMetrics {
	outcomeByMissionID := make(map[string]OutcomeEntry, len(outcomes))
	for _, o := range outcomes {
		outcomeByMissionID[o.MissionID] = o
	}

	var fallbackCount, fullPipelineCount, unnecessaryCount int
	for _, d := range decisions {
		if d.SelectedRoute == d.FallbackRoute && d.FallbackRoute != "" {
			fallbackCount++
		}
		if d.SelectedRoute != "full_pipeline" {
			continue
		}
		fullPipelineCount++
		if o, ok := outcomeByMissionID[d.MissionID]; ok && o.Status == analysisDeliveredStatus {
			unnecessaryCount++
		}
	}

	m := RouteMetrics{
		SampleSize:             len(decisions),
		FullPipelineSampleSize: fullPipelineCount,
	}
	if len(decisions) > 0 {
		m.FallbackRate = float64(fallbackCount) / float64(len(decisions))
	}
	if fullPipelineCount > 0 {
		m.UnnecessaryPipelineRate = float64(unnecessaryCount) / float64(fullPipelineCount)
	}
	return m
}

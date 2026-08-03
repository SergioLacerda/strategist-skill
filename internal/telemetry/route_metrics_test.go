package telemetry

import "testing"

func routeDecisionFixture(missionID, selectedRoute, fallbackRoute string) RouteDecision {
	return RouteDecision{
		MissionID:       missionID,
		RequestCategory: "general",
		SelectedRoute:   selectedRoute,
		RouteReason:     "test fixture",
		RouteConfidence: 0.8,
		EvidenceState:   "explicit",
		FallbackRoute:   fallbackRoute,
		Timestamp:       "2026-08-03T00:00:00Z",
	}
}

func TestComputeRouteMetrics_EmptyInput(t *testing.T) {
	t.Parallel()
	m := ComputeRouteMetrics(nil, nil)
	if m.FallbackRate != 0 || m.UnnecessaryPipelineRate != 0 {
		t.Fatalf("expected zero rates for empty input, got %+v", m)
	}
	if m.SampleSize != 0 || m.FullPipelineSampleSize != 0 {
		t.Fatalf("expected zero sample sizes, got %+v", m)
	}
}

func TestComputeRouteMetrics_FallbackRate(t *testing.T) {
	t.Parallel()
	decisions := []RouteDecision{
		routeDecisionFixture("m-1", "full_pipeline", "full_pipeline"), // fell back
		routeDecisionFixture("m-2", "critical_hit", "full_pipeline"),  // narrow route, no fallback
		routeDecisionFixture("m-3", "full_pipeline", "full_pipeline"), // fell back
		routeDecisionFixture("m-4", "implementation_short_route", "full_pipeline"),
	}

	m := ComputeRouteMetrics(decisions, nil)
	if got, want := m.FallbackRate, 0.5; got != want {
		t.Fatalf("FallbackRate = %v, want %v", got, want)
	}
	if m.SampleSize != 4 {
		t.Fatalf("SampleSize = %d, want 4", m.SampleSize)
	}
}

func TestComputeRouteMetrics_UnnecessaryPipelineRate(t *testing.T) {
	t.Parallel()
	decisions := []RouteDecision{
		routeDecisionFixture("m-1", "full_pipeline", "full_pipeline"),
		routeDecisionFixture("m-2", "full_pipeline", "full_pipeline"),
		routeDecisionFixture("m-3", "full_pipeline", "full_pipeline"),
		routeDecisionFixture("m-4", "critical_hit", "full_pipeline"), // excluded: not full_pipeline
	}
	outcomes := []OutcomeEntry{
		{MissionID: "m-1", Status: "analysis_delivered", Timestamp: "t"},
		{MissionID: "m-2", Status: "documentation_applied", Timestamp: "t"},
		// m-3 has no outcome entry at all — must not be counted as unnecessary.
		{MissionID: "m-4", Status: "analysis_delivered", Timestamp: "t"},
	}

	m := ComputeRouteMetrics(decisions, outcomes)
	if got, want := m.FullPipelineSampleSize, 3; got != want {
		t.Fatalf("FullPipelineSampleSize = %d, want %d", got, want)
	}
	if got, want := m.UnnecessaryPipelineRate, 1.0/3.0; got != want {
		t.Fatalf("UnnecessaryPipelineRate = %v, want %v", got, want)
	}
}

func TestComputeRouteMetrics_NoFullPipelineDecisions(t *testing.T) {
	t.Parallel()
	decisions := []RouteDecision{
		routeDecisionFixture("m-1", "critical_hit", "full_pipeline"),
	}
	m := ComputeRouteMetrics(decisions, nil)
	if m.UnnecessaryPipelineRate != 0 {
		t.Fatalf("expected zero UnnecessaryPipelineRate with no full_pipeline decisions, got %v", m.UnnecessaryPipelineRate)
	}
	if m.FullPipelineSampleSize != 0 {
		t.Fatalf("expected zero FullPipelineSampleSize, got %d", m.FullPipelineSampleSize)
	}
}

func TestComputeRouteMetrics_EmptyFallbackRouteNeverCountsAsFallback(t *testing.T) {
	t.Parallel()
	decisions := []RouteDecision{
		{MissionID: "m-1", SelectedRoute: "", FallbackRoute: ""},
	}
	m := ComputeRouteMetrics(decisions, nil)
	if m.FallbackRate != 0 {
		t.Fatalf("expected zero FallbackRate when fallback_route is empty, got %v", m.FallbackRate)
	}
}

package telemetry

import "testing"

func TestComputeFallbackMetrics_EmptySampleReturnsZeroNotNaN(t *testing.T) {
	t.Parallel()
	m := ComputeFallbackMetrics(nil)
	if m.AutoNativeRate != 0 || m.AskConfirmedRate != 0 || m.SampleSize != 0 {
		t.Fatalf("expected all-zero metrics for empty sample, got %+v", m)
	}
}

func TestComputeFallbackMetrics_MixedSample(t *testing.T) {
	t.Parallel()
	decisions := []FallbackDecision{
		{Outcome: "auto_native"},
		{Outcome: "auto_native"},
		{Outcome: "auto_native"},
		{Outcome: "ask_required"},
	}
	m := ComputeFallbackMetrics(decisions)
	if m.SampleSize != 4 {
		t.Fatalf("expected sample_size=4, got %d", m.SampleSize)
	}
	if got, want := m.AutoNativeRate, 0.75; got != want {
		t.Fatalf("AutoNativeRate: got %v, want %v", got, want)
	}
	if got, want := m.AskConfirmedRate, 0.25; got != want {
		t.Fatalf("AskConfirmedRate: got %v, want %v", got, want)
	}
}

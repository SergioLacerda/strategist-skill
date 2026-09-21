package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func gtLabel(mission, subject, label string) GroundTruthLabel {
	return GroundTruthLabel{
		MissionID: mission, Subject: subject, Label: label,
		Kind: domain.GroundTruthUserRevision, Ref: "review-" + mission, Timestamp: "2026-09-20T10:00:00Z",
	}
}

func TestValidateGroundTruthLabelRejectsUnsourcedAndUnknown(t *testing.T) {
	t.Parallel()
	require.NoError(t, ValidateGroundTruthLabel(gtLabel("m1", GroundTruthSubjectRoute, RouteLabelConfirmed)))
	err := ValidateGroundTruthLabel(GroundTruthLabel{Subject: "nope", Label: "x", Kind: "guess", Timestamp: "yesterday"})
	for _, want := range []string{"mission_id", "ground_truth_ref", "ground_truth_kind", "subject", "RFC3339"} {
		require.ErrorContains(t, err, want)
	}
	wrong := gtLabel("m1", GroundTruthSubjectRoute, HandoffApplicationLabelApplied)
	require.ErrorContains(t, ValidateGroundTruthLabel(wrong), "not allowed for subject")
}

func TestAppendAndReadGroundTruthLabelsAreIdempotentAndFiltered(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "memory", "labels.jsonl")
	got, err := ReadGroundTruthLabels(path, "")
	require.NoError(t, err)
	require.Nil(t, got, "missing file")
	_, err = AppendGroundTruthLabel(path, GroundTruthLabel{})
	require.Error(t, err, "expected validation error")

	require.True(t, appendLabel(t, path, GroundTruthSubjectRoute, RouteLabelConfirmed))
	require.False(t, appendLabel(t, path, GroundTruthSubjectRoute, RouteLabelReversed), "replay must not append")
	require.True(t, appendLabel(t, path, GroundTruthSubjectHandoffApplication, HandoffApplicationLabelApplied), "different subject must append")

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	require.NoError(t, err)
	_, _ = f.WriteString("not json\n{\"mission_id\":\"x\"}\n")
	require.NoError(t, f.Close())

	all, _ := ReadGroundTruthLabels(path, "")
	route, _ := ReadGroundTruthLabels(path, GroundTruthSubjectRoute)
	require.Len(t, all, 2)
	require.Len(t, route, 1)
	require.Equal(t, RouteLabelConfirmed, route[0].Label)
}

func appendLabel(t *testing.T, path, subject, label string) bool {
	t.Helper()
	ok, err := AppendGroundTruthLabel(path, gtLabel("m1", subject, label))
	require.NoError(t, err)
	return ok
}

func TestAppendGroundTruthLabelConcurrentReplayDoesNotInflate(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "labels.jsonl")
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = AppendGroundTruthLabel(path, gtLabel("m1", GroundTruthSubjectRoute, RouteLabelConfirmed))
		}()
	}
	wg.Wait()
	if got, _ := ReadGroundTruthLabels(path, ""); len(got) != 1 {
		t.Fatalf("labels = %d, want 1", len(got))
	}
}

func TestAppendGroundTruthLabelPathErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if _, err := AppendGroundTruthLabel(dir, gtLabel("m", GroundTruthSubjectRoute, RouteLabelConfirmed)); err == nil {
		t.Fatal("directory path must fail")
	}
	if _, err := ReadGroundTruthLabels(dir, ""); err == nil {
		t.Fatal("directory read must fail")
	}
	file := filepath.Join(dir, "f")
	_ = os.WriteFile(file, nil, 0o600)
	if _, err := AppendGroundTruthLabel(filepath.Join(file, "sub", "l.jsonl"), gtLabel("m", GroundTruthSubjectRoute, RouteLabelConfirmed)); err == nil {
		t.Fatal("unusable parent must fail")
	}
	if !strings.HasSuffix(GroundTruthLabelHistoryPath("/r"), filepath.FromSlash("memory/ground-truth-labels.jsonl")) {
		t.Fatal("history path")
	}
}

func TestComputeRouteGroundTruthMetrics(t *testing.T) {
	t.Parallel()
	empty := ComputeRouteGroundTruthMetrics([]RouteDecision{{MissionID: "a", SelectedRoute: "full_pipeline"}}, nil)
	if empty.SampleSize != 0 || empty.CalibrationStatus != domain.CalibrationNoSample || empty.RouteAccuracy != 0 {
		t.Fatalf("empty = %+v", empty)
	}
	decisions := []RouteDecision{
		{MissionID: "a", SelectedRoute: "full_pipeline"},
		{MissionID: "b", SelectedRoute: "critical_hit"},
		{MissionID: "c", SelectedRoute: "implementation_short_route"},
		{MissionID: "d", SelectedRoute: "full_pipeline"},
		{MissionID: "unlabeled", SelectedRoute: "full_pipeline"},
	}
	labels := []GroundTruthLabel{
		gtLabel("a", GroundTruthSubjectRoute, RouteLabelConfirmed),
		gtLabel("b", GroundTruthSubjectRoute, RouteLabelReversed),
		gtLabel("c", GroundTruthSubjectRoute, RouteLabelRiskUnderclassified),
		gtLabel("d", GroundTruthSubjectRoute, RouteLabelUserOverride),
		gtLabel("a", GroundTruthSubjectHandoffApplication, HandoffApplicationLabelApplied),
	}
	m := ComputeRouteGroundTruthMetrics(decisions, labels)
	if m.SampleSize != 4 || m.DirectRouteSampleSize != 2 || m.RouteAccuracy != 0.25 ||
		m.DirectRouteReversalRate != 0.5 || m.RiskUnderclassificationRate != 0.25 || m.UserOverrideRate != 0.25 ||
		m.CalibrationStatus != domain.CalibrationObserved {
		t.Fatalf("metrics = %+v", m)
	}
	small := ComputeRouteGroundTruthMetrics(decisions[:1], labels[:1])
	if small.CalibrationStatus != domain.CalibrationUncalibrated {
		t.Fatalf("small = %+v", small)
	}
}

func TestApplyHandoffApplicationGroundTruth(t *testing.T) {
	t.Parallel()
	base := HandoffMetrics{SampleSize: 3}
	none := ApplyHandoffApplicationGroundTruth(base, []GroundTruthLabel{gtLabel("a", GroundTruthSubjectRoute, RouteLabelConfirmed)})
	if none.ApplicationSampleSize != 0 || none.SemanticLoss.Application != 0 {
		t.Fatalf("none = %+v", none)
	}
	got := ApplyHandoffApplicationGroundTruth(base, []GroundTruthLabel{
		gtLabel("a", GroundTruthSubjectHandoffApplication, HandoffApplicationLabelApplied),
		gtLabel("b", GroundTruthSubjectHandoffApplication, HandoffApplicationLabelMissed),
		gtLabel("c", GroundTruthSubjectHandoffApplication, HandoffApplicationLabelApplied),
		gtLabel("d", GroundTruthSubjectHandoffApplication, HandoffApplicationLabelMissed),
	})
	if got.ApplicationSampleSize != 4 || got.SemanticLoss.Application != 0.5 {
		t.Fatalf("got = %+v", got)
	}
}

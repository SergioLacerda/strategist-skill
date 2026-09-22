package telemetry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestRecordGateOutcomeIsHumanGroundTruthOnly(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	got, err := GateOutcomeFor(root, "m1")
	require.NoError(t, err)
	require.Empty(t, got, "unanswered gate")

	_, err = RecordGateOutcome(root, "m1", "maybe", "gate-1")
	require.Error(t, err, "unknown outcome must be rejected")
	_, err = RecordGateOutcome(root, "m1", GateOutcomeAccepted, "")
	require.Error(t, err, "a gate outcome without a gate event reference must be rejected")

	appended, err := RecordGateOutcome(root, "m1", GateOutcomeRevisionRequested, "gate-1")
	require.NoError(t, err)
	require.True(t, appended)
	appended, _ = RecordGateOutcome(root, "m1", GateOutcomeAccepted, "gate-2")
	require.False(t, appended, "first human outcome must win")

	assertGateLabel(t, root, "m1", GateOutcomeRevisionRequested)
	other, _ := GateOutcomeFor(root, "other")
	require.Empty(t, other)
}

// assertGateLabel checks the mission's stored gate outcome and its provenance.
func assertGateLabel(t *testing.T, root, mission, want string) {
	t.Helper()
	got, _ := GateOutcomeFor(root, mission)
	require.Equal(t, want, got)
	labels, _ := ReadGroundTruthLabels(GroundTruthLabelHistoryPath(root), GroundTruthSubjectGateOutcome)
	require.Len(t, labels, 1)
	require.Equal(t, domain.GroundTruthUserRevision, labels[0].Kind)
	require.Equal(t, "approval_gate:gate-1", labels[0].Ref)
}

func TestGateOutcomeRejectsConfidenceProvenance(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	label := GroundTruthLabel{
		MissionID: "m1", Subject: GroundTruthSubjectGateOutcome,
		Label: GateOutcomeAccepted, Kind: domain.GroundTruthUserRevision,
		Ref: "confidence-records.jsonl:event-1", Timestamp: "2026-09-20T12:00:00Z",
	}
	if _, err := AppendGroundTruthLabel(GroundTruthLabelHistoryPath(root), label); err == nil {
		t.Fatal("confidence history must not be accepted as gate ground truth")
	}
}

func TestGateOutcomeErrorsPropagate(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.MkdirAll(GroundTruthLabelHistoryPath(root), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := GateOutcomeFor(root, "m1"); err == nil {
		t.Fatal("directory as history must fail")
	}
	if _, err := RecordGateOutcome(root, "m1", GateOutcomeAccepted, "g"); err == nil {
		t.Fatal("directory as history must fail")
	}
}

func TestLoadConfidenceGateReviewScopesByMission(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for _, mission := range []string{"m1", "m2"} {
		require.NoError(t, AppendMissingConfidenceRecord(ConfidenceHistoryPath(root), ConfidenceAgentScout, mission, "k", "not supplied", "2026-09-20T10:00:00Z"))
	}
	all, err := LoadConfidenceGateReview(root, "")
	require.NoError(t, err)
	require.Equal(t, 2, all.Metrics.MissingRecords)

	one, _ := LoadConfidenceGateReview(root, "m1")
	require.Equal(t, 1, one.Metrics.MissingRecords)
	require.True(t, one.ReviewRequired && one.Unavailable)
	none, _ := LoadConfidenceGateReview(root, "m3")
	require.True(t, none.Unavailable, "no records must be unavailable")

	bad := t.TempDir()
	require.NoError(t, os.MkdirAll(ConfidenceHistoryPath(bad), 0o755))
	_, err = LoadConfidenceGateReview(bad, "")
	require.Error(t, err, "unreadable history must fail")
}

func TestLoadConfidenceGateReviewForRunScopesExplicitRun(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := ConfidenceHistoryPath(root)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	for _, run := range []string{"first", "second"} {
		require.NoError(t, AppendMissingConfidenceRecordForRun(path, ConfidenceAgentRanger, "m1", run, "boundary", "not supplied", "2026-09-21T00:00:00Z"))
	}

	review, err := LoadConfidenceGateReviewForRun(root, "m1", "second")
	require.NoError(t, err)
	require.Equal(t, 1, review.Metrics.MissingRecords)
}

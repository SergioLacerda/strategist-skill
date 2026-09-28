package metrics

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testLedger = "role-levels.jsonl"

func testDependencies() Dependencies {
	return Dependencies{
		RootFlag: "root",
		ResolveRoot: func(_ *cobra.Command, _ string, explicit string) (string, error) {
			if explicit == "" {
				return "", fmt.Errorf("missing root")
			}
			return explicit, nil
		},
	}
}

func testCommand(use string) (*cobra.Command, *bytes.Buffer) {
	var out bytes.Buffer
	cmd := &cobra.Command{Use: use}
	cmd.SetOut(&out)
	return cmd, &out
}

func testRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	testutil.MinimalRoot(t, root)
	return root
}

func TestNewBuildsCompleteIndependentCommandTree(t *testing.T) {
	deps := testDependencies()
	cmd := New(deps, testLedger, 20)
	require.Len(t, cmd.Commands(), 10)
	for _, name := range []string{"handoff", "handoff-record", "confidence", "gate-outcome", "label", "levels", "mission-quality", "record", "rollout-check", "scout"} {
		found, _, err := cmd.Find([]string{name})
		require.NoError(t, err)
		assert.Equal(t, name, found.Name())
	}
	assert.NotSame(t, cmd, New(deps, testLedger, 20))
	root := &cobra.Command{Use: "root"}
	Register(root, deps, testLedger, 20)
	assert.Len(t, root.Commands(), 1)
}

func TestReadCommandsReportEmptyAndRecordedHistory(t *testing.T) {
	root := testRoot(t)
	deps := testDependencies()
	cmd, out := testCommand("metrics")
	require.NoError(t, RunHandoff(cmd, deps, root))
	assert.Contains(t, out.String(), "sample_size: 0")
	out.Reset()
	require.NoError(t, RunScout(cmd, deps, root))
	assert.Contains(t, out.String(), "fallback_rate: 0.00")

	mem := filepath.Join(root, "memory")
	require.NoError(t, os.MkdirAll(mem, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(mem, "handoff-challenges.jsonl"), []byte(`{"mission_id":"m","transition":"a","attempt":1,"status":"passed","passed":true}`+"\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(mem, "route-decisions.jsonl"), []byte(`{"mission_id":"m","selected_route":"full_pipeline","fallback_route":"full_pipeline"}`+"\n"), 0o600))
	out.Reset()
	require.NoError(t, RunHandoff(cmd, deps, root))
	assert.Contains(t, out.String(), "handoff_pass_rate: 1.00")
	out.Reset()
	require.NoError(t, RunScout(cmd, deps, root))
	assert.Contains(t, out.String(), "fallback_rate: 1.00")
}

func TestConfidenceRecordLabelAndGateCommands(t *testing.T) {
	root := testRoot(t)
	deps := testDependencies()
	cmd, out := testCommand("metrics")
	claim := filepath.Join(t.TempDir(), "claim.yaml")
	require.NoError(t, os.WriteFile(claim, []byte("claim:\n  id: C-1\n  statement: statement\n  agent: ranger\n  correlation_key: k\n  claim_kind: assertion\n  confidence_percent: 90\n  evidence_ids: [E-1]\n  evidence_classes: [explicit]\nevidence:\n  - {id: E-1, source_ref: docs/a.md, class: explicit, confidence: high}\n"), 0o600))
	require.NoError(t, RunRecord(cmd, deps, RecordOptions{Root: root, Mission: "m", Agent: "ranger", ClaimFile: claim}))
	assert.Contains(t, out.String(), "coverage_status: reported")
	out.Reset()
	require.NoError(t, RunRecord(cmd, deps, RecordOptions{Root: root, Mission: "m", Agent: "scout", Missing: true, CorrelationKey: "k", Reason: "missing"}))
	assert.Contains(t, out.String(), "missing-record recorded")

	out.Reset()
	require.NoError(t, RunGateOutcome(cmd, deps, root, "m", "accepted", "gate-1"))
	assert.Contains(t, out.String(), "gate outcome recorded")
	out.Reset()
	require.NoError(t, RunGateOutcome(cmd, deps, root, "m", "accepted", "gate-1"))
	assert.Contains(t, out.String(), "already recorded")
	out.Reset()
	require.NoError(t, RunLabel(cmd, deps, root, "m", telemetry.GroundTruthSubjectRoute, telemetry.RouteLabelReversed, "user_revision", "review-1"))
	assert.Contains(t, out.String(), "label recorded")
	out.Reset()
	require.NoError(t, RunLabel(cmd, deps, root, "m", telemetry.GroundTruthSubjectRoute, telemetry.RouteLabelReversed, "user_revision", "review-1"))
	assert.Contains(t, out.String(), "already recorded")
	out.Reset()
	require.NoError(t, RunConfidence(cmd, deps, root, "m"))
	assert.Contains(t, out.String(), "gate_outcome: accepted")
}

func TestLevelsAndRolloutCommands(t *testing.T) {
	root := testRoot(t)
	deps := testDependencies()
	path := filepath.Join(root, "memory", testLedger)
	require.NoError(t, leveling.AppendRecord(path, leveling.Record{MissionID: "m", Level: leveling.Level{Role: "ranger", Model: "Sonnet", Effort: "high", Source: "host"}}))
	cmd, out := testCommand("metrics")
	require.NoError(t, RunLevels(cmd, deps, LevelsOptions{Root: root, MaxRecords: 20}, testLedger))
	assert.Contains(t, out.String(), "role.ranger.records: 1")
	out.Reset()
	require.NoError(t, RunRollout(cmd, "advisory", ""))
	assert.Contains(t, out.String(), "enforcement advisory admissible")
	require.ErrorContains(t, RunRollout(cmd, "blocking", ""), "stays advisory")
	review := filepath.Join(t.TempDir(), "review.yaml")
	require.NoError(t, os.WriteFile(review, []byte("reviewed_by: r\nreviewed_at: 2026-09-20T12:00:00Z\ncompatibility_evidence_ref: e\nagent_denominators_reconciled: true\nsamples_per_agent: {scout: 3}\n"), 0o600))
	require.NoError(t, RunRollout(cmd, "blocking", review))
	require.Error(t, RunRollout(cmd, "blocking", filepath.Join(t.TempDir(), "absent.yaml")))
	for i := 0; i < 3; i++ {
		require.NoError(t, leveling.AppendRecord(path, leveling.Record{MissionID: "other", Level: leveling.Level{Role: "scout", Model: "Haiku", Effort: "low", Source: "policy"}}))
	}
	out.Reset()
	require.NoError(t, WriteLevelsReport(out, root, LevelsOptions{Mission: "m", Rotate: true, MaxRecords: 2}, testLedger))
	assert.Contains(t, out.String(), "rotated:")
	out.Reset()
	require.NoError(t, WriteLevelsReport(out, root, LevelsOptions{Mission: "m", JSON: true}, testLedger))
	assert.Contains(t, out.String(), "\"records\"")
}

func TestMetricsErrorsAreScoped(t *testing.T) {
	deps := testDependencies()
	cmd, _ := testCommand("metrics")
	require.Error(t, RunHandoff(cmd, deps, ""))
	require.ErrorContains(t, RunRecord(cmd, deps, RecordOptions{Root: testRoot(t), Mission: "m", Agent: "scout"}), "claim-file")
	require.ErrorContains(t, RunRecord(cmd, deps, RecordOptions{Root: testRoot(t), Mission: "m", Agent: "scout", Missing: true}), "correlation-key")
	require.ErrorContains(t, RunRecord(cmd, deps, RecordOptions{Root: testRoot(t), Mission: "m", Agent: "nobody", Missing: true, CorrelationKey: "k", Reason: "r"}), "unsupported agent")
	require.ErrorContains(t, RunGateOutcome(cmd, deps, testRoot(t), "m", "auto", "ref"), "metrics gate-outcome")
	require.ErrorContains(t, RunLabel(cmd, deps, testRoot(t), "m", telemetry.GroundTruthSubjectGateOutcome, telemetry.GateOutcomeAccepted, "user_revision", "ref"), "metrics label")
	bad := filepath.Join(t.TempDir(), "not-a-root")
	require.NoError(t, os.WriteFile(bad, []byte("x"), 0o600))
	require.ErrorContains(t, RunScout(cmd, deps, bad), "metrics scout")
	require.ErrorContains(t, RunConfidence(cmd, deps, bad, ""), "metrics confidence")
	require.ErrorContains(t, RunLevels(cmd, deps, LevelsOptions{Root: bad}, testLedger), "metrics levels")
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, fmt.Errorf("write failed") }

func TestMetricRenderersPropagateWriterErrors(t *testing.T) {
	w := failingWriter{}
	require.Error(t, PrintHandoffMetrics(w, telemetry.HandoffMetrics{}))
	require.Error(t, PrintRouteMetrics(w, telemetry.RouteMetrics{}))
	require.Error(t, PrintRouteGroundTruthMetrics(w, telemetry.RouteGroundTruthMetrics{}))
	require.Error(t, PrintConfidenceMetrics(w, telemetry.ConfidenceGateReview{}))
	require.Error(t, PrintGateOutcome(w, testRoot(t), "m"))
	require.Error(t, WriteLevelsReport(w, testRoot(t), LevelsOptions{}, testLedger))
}

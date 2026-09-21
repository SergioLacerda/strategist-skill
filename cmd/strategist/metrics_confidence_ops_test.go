package main

import (
	"os"
	"runtime"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireFlagsPanicsOnUnknownFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "x"}
	cmd.Flags().String("known", "", "")
	assert.NotPanics(t, func() { requireFlags(cmd, "known") })
	assert.Panics(t, func() { requireFlags(cmd, "unknown") })
}

func TestMetricsCommandsRejectMissingRoot(t *testing.T) {
	t.Chdir(t.TempDir()) // no .strategist here, so root discovery fails
	missing := ""
	require.Error(t, runMetricsRecord(metricsRecordCmd, metricsRecordOptions{Root: missing, Mission: "m", Agent: "scout", Missing: true, CorrelationKey: "k", Reason: "r"}))
	require.Error(t, runMetricsGateOutcome(metricsGateOutcomeCmd, metricsGateOutcomeOptions{Root: missing, Mission: "m", Outcome: "accepted", Ref: "g"}))
	require.Error(t, runMetricsConfidence(metricsConfidenceCmd, metricsConfidenceOptions{Root: missing}))
}

func TestMetricsConfidenceMissionScopeAndReadErrors(t *testing.T) {
	root := t.TempDir()
	testutil.MinimalRoot(t, root)
	producer, err := telemetry.NewConfidenceProducerAdapter(telemetry.ConfidenceHistoryPath(root), telemetry.ConfidenceAgentScout, "m-1")
	require.NoError(t, err)
	require.NoError(t, producer.RecordMissing("k", "not supplied"))
	out := captureStdout(t, func() {
		require.NoError(t, runMetricsConfidence(metricsConfidenceCmd, metricsConfidenceOptions{Root: root, Mission: "m-1"}))
		require.NoError(t, runMetricsConfidence(metricsConfidenceCmd, metricsConfidenceOptions{Root: root}))
	})
	assert.Contains(t, out, "gate_outcome: none")

	if runtime.GOOS == "windows" {
		t.Skip("reading a directory as a file has different semantics on Windows")
	}
	bad := t.TempDir()
	testutil.MinimalRoot(t, bad)
	require.NoError(t, os.MkdirAll(telemetry.ConfidenceHistoryPath(bad), 0o755))
	require.Error(t, runMetricsConfidence(metricsConfidenceCmd, metricsConfidenceOptions{Root: bad}))
	require.Error(t, runMetricsRecord(metricsRecordCmd, metricsRecordOptions{Root: bad, Mission: "m", Agent: "scout", Missing: true, CorrelationKey: "k", Reason: "r"}))
	require.NoError(t, os.MkdirAll(telemetry.GroundTruthLabelHistoryPath(bad), 0o755))
	require.Error(t, printGateOutcome(&testBuffer{}, bad, "m"))
}

type testBuffer struct{}

func (*testBuffer) Write(p []byte) (int, error) { return len(p), nil }

func TestMetricsHandoffAndScoutSurfaceLabelReadErrors(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("reading a directory as a file has different semantics on Windows")
	}
	root := t.TempDir()
	testutil.MinimalRoot(t, root)
	require.NoError(t, os.MkdirAll(telemetry.GroundTruthLabelHistoryPath(root), 0o755))
	require.ErrorContains(t, runMetricsHandoff(metricsHandoffCmd, metricsHandoffOptions{Root: root}), "metrics handoff")
	require.ErrorContains(t, runMetricsScout(metricsScoutCmd, metricsScoutOptions{Root: root}), "metrics scout")
}
